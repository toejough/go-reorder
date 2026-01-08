package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIProcessesSingleFile(t *testing.T) {
	// Create temp file with unordered code
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}

const Version = "1.0"

type Config struct{}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run CLI
	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{inputFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Should output reordered code to stdout
	output := stdout.String()
	if !strings.Contains(output, "Version") {
		t.Errorf("expected output to contain reordered code, got: %s", output)
	}
}

func TestCLIWriteFlag(t *testing.T) {
	// Create temp file with unordered code
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}

const Version = "1.0"

type Config struct{}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run CLI with --write
	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--write", inputFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	// File should be modified
	modified, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	// Const should appear before func in reordered output
	constPos := strings.Index(string(modified), "const Version")
	funcPos := strings.Index(string(modified), "func Helper")
	if constPos > funcPos {
		t.Errorf("expected const before func in reordered file, got:\n%s", string(modified))
	}
}

func TestCLICheckFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Unordered file should return exit code 1
	t.Run("unordered file returns exit 1", func(t *testing.T) {
		inputFile := filepath.Join(tmpDir, "unordered.go")
		content := `package test

func Helper() {}

const Version = "1.0"
`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var stdout, stderr bytes.Buffer
		exitCode := runCLI([]string{"--check", inputFile}, &stdout, &stderr)

		if exitCode != 1 {
			t.Errorf("expected exit code 1 for unordered file, got %d", exitCode)
		}
	})

	// Already ordered file should return exit code 0
	t.Run("ordered file returns exit 0", func(t *testing.T) {
		inputFile := filepath.Join(tmpDir, "ordered.go")
		// Use exact output format that reorder produces
		content := `package test

// Exported constants.
const (
	Version = "1.0"
)

func Helper() {}
`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var stdout, stderr bytes.Buffer
		exitCode := runCLI([]string{"--check", inputFile}, &stdout, &stderr)

		if exitCode != 0 {
			t.Errorf("expected exit code 0 for ordered file, got %d; stderr: %s", exitCode, stderr.String())
		}
	})
}

func TestCLIDiffFlag(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}

const Version = "1.0"
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--diff", inputFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Should contain diff markers
	output := stdout.String()
	if !strings.Contains(output, "---") || !strings.Contains(output, "+++") {
		t.Errorf("expected diff output with --- and +++, got: %s", output)
	}
}

func TestCLIProcessesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple Go files
	file1 := filepath.Join(tmpDir, "a.go")
	file2 := filepath.Join(tmpDir, "b.go")

	content := `package test

func Helper() {}

const Version = "1.0"
`
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run CLI with --write on directory
	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--write", tmpDir}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	// Both files should be modified
	for _, f := range []string{file1, file2} {
		modified, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read modified file %s: %v", f, err)
		}
		constPos := strings.Index(string(modified), "const Version")
		funcPos := strings.Index(string(modified), "func Helper")
		if constPos > funcPos {
			t.Errorf("expected const before func in %s, got:\n%s", f, string(modified))
		}
	}
}

func TestCLIProcessesDirectoryRecursively(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	file1 := filepath.Join(tmpDir, "a.go")
	file2 := filepath.Join(subDir, "b.go")

	content := `package test

func Helper() {}

const Version = "1.0"
`
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run CLI with --write on directory
	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--write", tmpDir}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	// Both files should be modified (including nested)
	for _, f := range []string{file1, file2} {
		modified, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read modified file %s: %v", f, err)
		}
		constPos := strings.Index(string(modified), "const Version")
		funcPos := strings.Index(string(modified), "func Helper")
		if constPos > funcPos {
			t.Errorf("expected const before func in %s, got:\n%s", f, string(modified))
		}
	}
}

func TestCLIConfigFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file with custom order
	configFile := filepath.Join(tmpDir, "reorder.toml")
	configContent := `[behavior]
mode = "append"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create test file
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}

const Version = "1.0"
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run CLI with --config
	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--config", configFile, inputFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
}

func TestCLIModeFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config that excludes some sections (no uncategorized for drop test)
	dropConfigFile := filepath.Join(tmpDir, "drop.toml")
	dropConfigContent := `[sections]
order = ["imports", "main"]

[behavior]
mode = "strict"
`
	if err := os.WriteFile(dropConfigFile, []byte(dropConfigContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create config with uncategorized for append test
	appendConfigFile := filepath.Join(tmpDir, "append.toml")
	appendConfigContent := `[sections]
order = ["imports", "main", "uncategorized"]

[behavior]
mode = "strict"
`
	if err := os.WriteFile(appendConfigFile, []byte(appendConfigContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create test file with code that won't fit in the limited sections
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}

const Version = "1.0"
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("mode=drop discards unmatched", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runCLI([]string{"--config", dropConfigFile, "--mode", "drop", inputFile}, &stdout, &stderr)

		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
		}

		// Output should not contain Helper or Version (they were dropped)
		output := stdout.String()
		if strings.Contains(output, "Helper") || strings.Contains(output, "Version") {
			t.Errorf("expected dropped code to be removed, got: %s", output)
		}
	})

	t.Run("mode=append appends unmatched silently", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runCLI([]string{"--config", appendConfigFile, "--mode", "append", inputFile}, &stdout, &stderr)

		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
		}

		// Output should contain the appended code
		output := stdout.String()
		if !strings.Contains(output, "Helper") || !strings.Contains(output, "Version") {
			t.Errorf("expected appended code to be present, got: %s", output)
		}
	})
}

func TestCLIMissingConfigError(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--config", "/nonexistent/config.toml", inputFile}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing config, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "Error") {
		t.Errorf("expected error message in stderr, got: %s", stderr.String())
	}
}

func TestCLIShowsFileBeingProcessed(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Helper() {}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runCLI([]string{"--write", inputFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	// stderr should show which file was processed
	if !strings.Contains(stderr.String(), "test.go") {
		t.Errorf("expected stderr to show file being processed, got: %s", stderr.String())
	}
}
