package main

import (
	"io"
	"os"
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
