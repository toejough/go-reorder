package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/toejough/go-reorder"
)

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
	var configPath string
	if opts.config != "" {
		// Check if explicit config file exists
		if _, err := os.Stat(opts.config); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(stderr, "Error: config file not found: %s\n", opts.config)
			return 1
		}
		configPath = opts.config
		cfg, err = reorder.LoadConfig(opts.config)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error loading config: %v\n", err)
			return 1
		}
	} else {
		// Try to discover config based on first file's directory
		firstFileDir := filepath.Dir(goFiles[0])
		configPath, err = reorder.FindConfig(firstFileDir)
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

	// Verbose output
	if opts.verbose {
		if configPath != "" {
			_, _ = fmt.Fprintf(stderr, "config: %s\n", configPath)
		} else {
			_, _ = fmt.Fprintf(stderr, "config: using defaults\n")
		}
		_, _ = fmt.Fprintf(stderr, "mode: %s\n", cfg.Behavior.Mode)
		_, _ = fmt.Fprintf(stderr, "files: %d\n", len(goFiles))
	}

	// Process each file
	var changedFiles []string
	for _, f := range goFiles {
		changed, err := processFile(f, cfg, opts, stdout, stderr)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error processing %s: %v\n", f, err)
			return 1
		}
		if changed {
			changedFiles = append(changedFiles, f)
		}
	}

	// --check mode: exit 1 if any files would change
	if opts.check && len(changedFiles) > 0 {
		for _, f := range changedFiles {
			_, _ = fmt.Fprintf(stderr, "%s\n", f)
		}
		return 1
	}

	return 0
}
