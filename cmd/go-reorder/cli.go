package main

import (
	"fmt"
	"io"
	"os"
)

// CLI represents the go-reorder command.
type CLI struct {
	Write        bool     `targ:"flag,short=w,desc=Write result to source file instead of stdout"`
	Check        bool     `targ:"flag,short=c,desc=Check if files are properly ordered (exit 1 if not)"`
	Diff         bool     `targ:"flag,short=d,desc=Display diff instead of reordered source"`
	Verbose      bool     `targ:"flag,short=v,desc=Show config and processing details"`
	Init         bool     `targ:"flag,name=init,desc=Create a default .go-reorder.toml config file"`
	ListSections bool     `targ:"flag,name=list-sections,desc=List available section names for config"`
	Config       string   `targ:"flag,name=config,desc=Path to config file"`
	Mode         string   `targ:"flag,name=mode,desc=Behavior mode (strict|warn|append|drop)"`
	Exclude      []string `targ:"flag,name=exclude,desc=Exclude files matching pattern (can be repeated)"`
	Path         string   `targ:"positional,placeholder=PATH,desc=File or directory to process"`
}

// Reorder Go source files.
// Reorders declarations in Go files according to a configurable order.
func (c *CLI) Run() error {
	stdin := io.Reader(os.Stdin)
	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)

	if testCtx != nil {
		if testCtx.stdin != nil {
			stdin = testCtx.stdin
		}
		stdout = testCtx.stdout
		stderr = testCtx.stderr
	}

	// Handle --list-sections
	if c.ListSections {
		sections := []string{
			"imports", "main", "init",
			"exported_consts", "exported_enums", "exported_vars",
			"exported_types", "exported_funcs",
			"unexported_consts", "unexported_enums", "unexported_vars",
			"unexported_types", "unexported_funcs",
			"uncategorized",
		}
		_, _ = fmt.Fprintln(stdout, "Available sections for config:")
		for _, s := range sections {
			_, _ = fmt.Fprintf(stdout, "  %s\n", s)
		}
		return nil
	}

	// Handle --init
	if c.Init {
		exitCode := c.runInit(stdout, stderr)
		if testCtx != nil {
			testCtx.exitCode = exitCode
			return nil
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	}

	opts := cliOptions{
		write:   c.Write,
		check:   c.Check,
		diff:    c.Diff,
		verbose: c.Verbose,
		config:  c.Config,
		mode:    c.Mode,
		exclude: c.Exclude,
	}

	var files []string
	if c.Path != "" {
		files = []string{c.Path}
	}

	exitCode := run(opts, files, stdin, stdout, stderr)

	if testCtx != nil {
		testCtx.exitCode = exitCode
		return nil
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// unexported variables.
var (
	testCtx *testContext
)

type cliOptions struct {
	write   bool
	check   bool
	diff    bool
	verbose bool
	config  string
	mode    string
	exclude []string
}

// testContext holds test injection - separate from CLI to avoid targ's zero-value check.
type testContext struct {
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	exitCode int
}

// runInit creates a default .go-reorder.toml config file.
// Returns exit code (0 for success, 1 for error).
func (c *CLI) runInit(stdout, stderr io.Writer) int {
	configPath := ".go-reorder.toml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s already exists\n", configPath)
		return 1
	}

	configContent := `# go-reorder configuration
# See https://github.com/toejough/go-reorder for documentation

[sections]
# Order of declaration sections in each file
# Remove sections you don't want, or reorder as needed
order = [
  "imports",
  "main",
  "init",
  "exported_consts",
  "exported_enums",
  "exported_vars",
  "exported_types",
  "exported_funcs",
  "unexported_consts",
  "unexported_enums",
  "unexported_vars",
  "unexported_types",
  "unexported_funcs",
  "uncategorized",
]

[types]
# How to order elements within a type group
type_layout = ["typedef", "constructors", "exported_methods", "unexported_methods"]

# How to order elements within an enum group
enum_layout = ["typedef", "iota", "exported_methods", "unexported_methods"]

[behavior]
# strict: Error if code has no matching section (default)
# warn:   Append unmatched code at end with warning
# append: Silently append unmatched code at end
# drop:   Discard unmatched code (dangerous!)
mode = "strict"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error writing config: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Created %s\n", configPath)
	return 0
}
