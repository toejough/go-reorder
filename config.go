package reorder

import (
	"errors"
	"fmt"
)

// ValidSections defines all recognized section names.
var ValidSections = map[string]bool{
	"imports":            true,
	"main":               true,
	"init":               true,
	"exported_consts":    true,
	"exported_enums":     true,
	"exported_vars":      true,
	"exported_types":     true,
	"exported_funcs":     true,
	"unexported_consts":  true,
	"unexported_enums":   true,
	"unexported_vars":    true,
	"unexported_types":   true,
	"unexported_funcs":   true,
	"uncategorized":      true,
}

// ValidModes defines all recognized behavior modes.
var ValidModes = map[string]bool{
	"strict": true,
	"warn":   true,
	"append": true,
	"drop":   true,
}

// Config holds all configuration for go-reorder.
type Config struct {
	Sections SectionsConfig
	Behavior BehaviorConfig
}

// SectionsConfig controls declaration ordering.
type SectionsConfig struct {
	Order []string
}

// BehaviorConfig controls how the reorderer handles edge cases.
type BehaviorConfig struct {
	Mode string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Sections: SectionsConfig{
			Order: []string{
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
			},
		},
		Behavior: BehaviorConfig{
			Mode: "strict",
		},
	}
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	seen := make(map[string]bool)

	for _, section := range c.Sections.Order {
		if !ValidSections[section] {
			return fmt.Errorf("unknown section: %q", section)
		}
		if seen[section] {
			return fmt.Errorf("duplicate section: %q", section)
		}
		seen[section] = true
	}

	if !ValidModes[c.Behavior.Mode] {
		return fmt.Errorf("unknown mode: %q (valid: strict, warn, append, drop)", c.Behavior.Mode)
	}

	return nil
}

// ErrInvalidConfig is returned when config validation fails.
var ErrInvalidConfig = errors.New("invalid config")