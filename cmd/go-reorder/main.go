package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/toejough/go-reorder"
	"github.com/toejough/targ"
)

// CLI represents the go-reorder command.
type CLI struct {
	Write   bool     `targ:"flag,short=w,desc=Write result to source file instead of stdout"`
	Check   bool     `targ:"flag,short=c,desc=Check if files are properly ordered (exit 1 if not)"`
	Diff    bool     `targ:"flag,short=d,desc=Display diff instead of reordered source"`
	Config  string   `targ:"flag,name=config,desc=Path to config file"`
	Mode    string   `targ:"flag,name=mode,desc=Behavior mode (strict|warn|append|drop)"`
	Exclude []string `targ:"flag,name=exclude,desc=Exclude files matching pattern (can be repeated)"`
	Path    string   `targ:"positional,placeholder=PATH,desc=File or directory to process"`
}

// Reorder Go source files.
// Reorders declarations in Go files according to a configurable order.
func (c *CLI) Run() error {
	opts := cliOptions{
		write:   c.Write,
		check:   c.Check,
		diff:    c.Diff,
		config:  c.Config,
		mode:    c.Mode,
		exclude: c.Exclude,
	}

	var files []string
	if c.Path != "" {
		files = []string{c.Path}
	}

	exitCode := run(opts, files, os.Stdin, os.Stdout, os.Stderr)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

func main() {
	targ.Run(CLI{})
}

// runCLI is the testable entry point (parses args).
func runCLI(args []string, stdout, stderr io.Writer) int {
	return runCLIWithStdin(args, nil, stdout, stderr)
}

// runCLIWithStdin is the testable entry point with stdin support (parses args).
func runCLIWithStdin(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, files, err := parseArgs(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return run(opts, files, stdin, stdout, stderr)
}

// run is the core logic, taking already-parsed options.
func run(opts cliOptions, files []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(files) == 0 {
		_, _ = fmt.Fprintf(stderr, "Error: no files specified\n")
		return 1
	}

	// Handle stdin mode
	if len(files) == 1 && files[0] == "-" {
		return processStdin(stdin, opts, stdout, stderr)
	}

	// Discover all Go files first (needed for config discovery)
	goFiles, err := discoverFiles(files, opts.exclude)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error discovering files: %v\n", err)
		return 1
	}

	if len(goFiles) == 0 {
		_, _ = fmt.Fprintf(stderr, "Error: no Go files found\n")
		return 1
	}

	// Load config
	var cfg *reorder.Config
	if opts.config != "" {
		// Check if explicit config file exists
		if _, err := os.Stat(opts.config); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(stderr, "Error: config file not found: %s\n", opts.config)
			return 1
		}
		cfg, err = reorder.LoadConfig(opts.config)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error loading config: %v\n", err)
			return 1
		}
	} else {
		// Try to discover config based on first file's directory
		firstFileDir := filepath.Dir(goFiles[0])
		configPath, err := reorder.FindConfig(firstFileDir)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error finding config: %v\n", err)
			return 1
		}
		if configPath != "" {
			cfg, err = reorder.LoadConfig(configPath)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "Error loading config: %v\n", err)
				return 1
			}
		} else {
			cfg = reorder.DefaultConfig()
		}
	}

	// Override mode if specified via flag
	if opts.mode != "" {
		cfg.Behavior.Mode = opts.mode
	}

	// Process each file
	hasChanges := false
	for _, f := range goFiles {
		changed, err := processFile(f, cfg, opts, stdout, stderr)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error processing %s: %v\n", f, err)
			return 1
		}
		if changed {
			hasChanges = true
		}
	}

	// --check mode: exit 1 if any files would change
	if opts.check && hasChanges {
		return 1
	}

	return 0
}

type cliOptions struct {
	write   bool
	check   bool
	diff    bool
	config  string
	mode    string
	exclude []string
}

func parseArgs(args []string) (cliOptions, []string, error) {
	var opts cliOptions
	var files []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-w" || arg == "--write":
			opts.write = true
		case arg == "-c" || arg == "--check":
			opts.check = true
		case arg == "-d" || arg == "--diff":
			opts.diff = true
		case arg == "--config":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--config requires a value")
			}
			i++
			opts.config = args[i]
		case strings.HasPrefix(arg, "--config="):
			opts.config = strings.TrimPrefix(arg, "--config=")
		case arg == "--mode":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--mode requires a value")
			}
			i++
			opts.mode = args[i]
		case strings.HasPrefix(arg, "--mode="):
			opts.mode = strings.TrimPrefix(arg, "--mode=")
		case arg == "--exclude":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--exclude requires a value")
			}
			i++
			opts.exclude = append(opts.exclude, args[i])
		case strings.HasPrefix(arg, "--exclude="):
			opts.exclude = append(opts.exclude, strings.TrimPrefix(arg, "--exclude="))
		case strings.HasPrefix(arg, "-") && arg != "-":
			return opts, nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			files = append(files, arg)
		}
	}

	return opts, files, nil
}

func discoverFiles(paths []string, excludePatterns []string) ([]string, error) {
	var files []string

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			// Single file
			if strings.HasSuffix(p, ".go") {
				if !isExcluded(p, excludePatterns) {
					files = append(files, p)
				}
			}
			continue
		}

		// Directory: walk recursively
		baseDir := p
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				// Get relative path for pattern matching
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					relPath = path
				}
				if !isExcluded(relPath, excludePatterns) {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// isExcluded checks if a path matches any of the exclude patterns.
func isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check the base name for patterns like "*_test.go"
		if matched, err := doublestar.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
	}
	return false
}

// processStdin handles reading from stdin and writing to stdout.
func processStdin(stdin io.Reader, opts cliOptions, stdout, stderr io.Writer) int {
	// Read all from stdin
	content, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error reading stdin: %v\n", err)
		return 1
	}

	// Load config
	var cfg *reorder.Config
	if opts.config != "" {
		if _, err := os.Stat(opts.config); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(stderr, "Error: config file not found: %s\n", opts.config)
			return 1
		}
		cfg, err = reorder.LoadConfig(opts.config)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error loading config: %v\n", err)
			return 1
		}
	} else {
		cfg = reorder.DefaultConfig()
	}

	// Override mode if specified via flag
	if opts.mode != "" {
		cfg.Behavior.Mode = opts.mode
	}

	// Reorder
	result, err := reorder.SourceWithConfig(string(content), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	// Output to stdout
	_, _ = fmt.Fprint(stdout, result)
	return 0
}

func processFile(path string, cfg *reorder.Config, opts cliOptions, stdout, stderr io.Writer) (bool, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Reorder
	result, err := reorder.SourceWithConfig(string(content), cfg)
	if err != nil {
		return false, err
	}

	// Check if changed
	changed := result != string(content)

	// Handle output based on flags
	if opts.check {
		// Just check, don't output anything
		return changed, nil
	}

	if opts.diff {
		if changed {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(content)),
				B:        difflib.SplitLines(result),
				FromFile: path,
				ToFile:   path,
				Context:  3,
			}
			text, err := difflib.GetUnifiedDiffString(diff)
			if err != nil {
				return false, err
			}
			_, _ = fmt.Fprint(stdout, text)
		}
		return changed, nil
	}

	if opts.write {
		_, _ = fmt.Fprintf(stderr, "%s\n", path)
		if changed {
			if err := os.WriteFile(path, []byte(result), 0644); err != nil {
				return false, err
			}
		}
		return changed, nil
	}

	// Default: output to stdout
	_, _ = fmt.Fprint(stdout, result)
	return changed, nil
}
