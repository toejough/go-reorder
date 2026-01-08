package main

import (
	"io"

	"github.com/toejough/targ"
)

// CLI represents the go-reorder command.
type CLI struct {
	Write  bool   `targ:"flag,short=w,desc=Write result to source file instead of stdout"`
	Check  bool   `targ:"flag,short=c,desc=Check if files are properly ordered (exit 1 if not)"`
	Diff   bool   `targ:"flag,short=d,desc=Display diff instead of reordered source"`
	Config string `targ:"flag,name=config,desc=Path to config file"`
	Files  string `targ:"positional,placeholder=FILES,desc=Files or directories to process"`
}

// Reorder Go source files.
// Reorders declarations in Go files according to a configurable order.
func (c *CLI) Run() error {
	return nil
}

func main() {
	targ.Run(CLI{})
}

// runCLI is the testable entry point.
func runCLI(args []string, stdout, stderr io.Writer) int {
	// TODO: Implement
	return 1
}
