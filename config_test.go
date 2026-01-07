package reorder

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

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
		cfg := DefaultConfig()
		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("unknown section errors", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Sections.Order = []string{"bogus"}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for unknown section")
		}
	})

	t.Run("duplicate sections error", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Sections.Order = []string{"imports", "imports"}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for duplicate sections")
		}
	})

	t.Run("invalid mode errors", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Behavior.Mode = "invalid"
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid mode")
		}
	})
}