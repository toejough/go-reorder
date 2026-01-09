package reorder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/go-reorder"
)

func TestDefaultConfig(t *testing.T) {
	cfg := reorder.DefaultConfig()

	if len(cfg.Sections.Order) != 14 {
		t.Errorf("expected 14 sections, got %d", len(cfg.Sections.Order))
	}
	if len(cfg.Sections.Order) > 0 && cfg.Sections.Order[0] != "imports" {
		t.Errorf("expected first section to be imports, got %q", cfg.Sections.Order[0])
	}
	if len(cfg.Sections.Order) > 1 && cfg.Sections.Order[1] != "main" {
		t.Errorf("expected second section to be main, got %q", cfg.Sections.Order[1])
	}
	if len(cfg.Sections.Order) > 2 && cfg.Sections.Order[2] != "init" {
		t.Errorf("expected third section to be init, got %q", cfg.Sections.Order[2])
	}
	if len(cfg.Sections.Order) > 13 && cfg.Sections.Order[13] != "uncategorized" {
		t.Errorf("expected last section to be uncategorized, got %q", cfg.Sections.Order[13])
	}
	if cfg.Behavior.Mode != "strict" {
		t.Errorf("expected mode to be strict, got %q", cfg.Behavior.Mode)
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := reorder.DefaultConfig()
		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("unknown section errors", func(t *testing.T) {
		cfg := reorder.DefaultConfig()
		cfg.Sections.Order = []string{"bogus"}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for unknown section")
		}
	})

	t.Run("duplicate sections error", func(t *testing.T) {
		cfg := reorder.DefaultConfig()
		cfg.Sections.Order = []string{"imports", "imports"}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for duplicate sections")
		}
	})

	t.Run("invalid mode errors", func(t *testing.T) {
		cfg := reorder.DefaultConfig()
		cfg.Behavior.Mode = "invalid"
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid mode")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid TOML file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		content := `
[behavior]
mode = "warn"

[sections]
order = ["imports", "main", "uncategorized"]
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := reorder.LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Behavior.Mode != "warn" {
			t.Errorf("expected mode warn, got %q", cfg.Behavior.Mode)
		}
		if len(cfg.Sections.Order) != 3 {
			t.Errorf("expected 3 sections, got %d", len(cfg.Sections.Order))
		}
	})

	t.Run("missing file returns defaults", func(t *testing.T) {
		cfg, err := reorder.LoadConfig("/nonexistent/path/config.toml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Behavior.Mode != "strict" {
			t.Errorf("expected default mode strict, got %q", cfg.Behavior.Mode)
		}
		if len(cfg.Sections.Order) != 14 {
			t.Errorf("expected 14 default sections, got %d", len(cfg.Sections.Order))
		}
	})

	t.Run("partial config merges with defaults", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		content := `
[behavior]
mode = "append"
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := reorder.LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Behavior.Mode != "append" {
			t.Errorf("expected mode append, got %q", cfg.Behavior.Mode)
		}
		// Sections should keep defaults when not specified
		if len(cfg.Sections.Order) != 14 {
			t.Errorf("expected 14 default sections, got %d", len(cfg.Sections.Order))
		}
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		content := `this is not valid toml {{{`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := reorder.LoadConfig(path)
		if err == nil {
			t.Error("expected error for invalid TOML")
		}
	})

	t.Run("invalid values return error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		content := `
[behavior]
mode = "bogus_mode"
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := reorder.LoadConfig(path)
		if err == nil {
			t.Error("expected error for invalid mode value")
		}
	})
}

func TestFindConfig(t *testing.T) {
	t.Run("finds config in current directory", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".go-reorder.toml")
		if err := os.WriteFile(configPath, []byte("[behavior]\nmode = \"warn\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != configPath {
			t.Errorf("expected %q, got %q", configPath, found)
		}
	})

	t.Run("finds config in parent directory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "sub")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(dir, ".go-reorder.toml")
		if err := os.WriteFile(configPath, []byte("[behavior]\nmode = \"warn\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != configPath {
			t.Errorf("expected %q, got %q", configPath, found)
		}
	})

	t.Run("stops at .git boundary", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "repo", "sub")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create .git in repo/
		gitDir := filepath.Join(dir, "repo", ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Put config above .git (should not be found)
		configPath := filepath.Join(dir, ".go-reorder.toml")
		if err := os.WriteFile(configPath, []byte("[behavior]\nmode = \"warn\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != "" {
			t.Errorf("expected empty (stopped at .git), got %q", found)
		}
	})

	t.Run("stops at go.mod boundary", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "module", "sub")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create go.mod in module/
		goModPath := filepath.Join(dir, "module", "go.mod")
		if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Put config above go.mod (should not be found)
		configPath := filepath.Join(dir, ".go-reorder.toml")
		if err := os.WriteFile(configPath, []byte("[behavior]\nmode = \"warn\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != "" {
			t.Errorf("expected empty (stopped at go.mod), got %q", found)
		}
	})

	t.Run("returns empty when no config found", func(t *testing.T) {
		dir := t.TempDir()
		// Create .git to establish boundary
		gitDir := filepath.Join(dir, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != "" {
			t.Errorf("expected empty, got %q", found)
		}
	})

	t.Run("finds config at boundary directory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "repo", "sub")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create .git in repo/
		gitDir := filepath.Join(dir, "repo", ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Put config at the .git level (should be found)
		configPath := filepath.Join(dir, "repo", ".go-reorder.toml")
		if err := os.WriteFile(configPath, []byte("[behavior]\nmode = \"warn\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := reorder.FindConfig(subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != configPath {
			t.Errorf("expected %q, got %q", configPath, found)
		}
	})
}
