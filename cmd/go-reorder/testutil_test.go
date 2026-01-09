package main

import (
	"io"

	"github.com/toejough/targ"
)

// executeCLI is the testable entry point using targ.Execute.
func executeCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	testCtx = &testContext{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
	defer func() { testCtx = nil }()

	_, _ = targ.Execute(append([]string{"go-reorder"}, args...), CLI{})
	return testCtx.exitCode
}
