package reorder

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
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

// ValidTypeLayoutElements defines valid elements for type layout.
var ValidTypeLayoutElements = map[string]bool{
	"typedef":            true,
	"constructors":       true,
	"exported_methods":   true,
	"unexported_methods": true,
}

// ValidEnumLayoutElements defines valid elements for enum layout.
var ValidEnumLayoutElements = map[string]bool{
	"typedef":            true,
	"iota":               true,
	"exported_methods":   true,
	"unexported_methods": true,
}

// Config holds all configuration for go-reorder.
type Config struct {
	Sections SectionsConfig
	Types    TypesConfig
	Behavior BehaviorConfig
}

// TypesConfig controls how types and enums are laid out internally.
type TypesConfig struct {
	TypeLayout []string
	EnumLayout []string
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
		Types: TypesConfig{
			TypeLayout: []string{
				"typedef",
				"constructors",
				"exported_methods",
				"unexported_methods",
			},
			EnumLayout: []string{
				"typedef",
				"iota",
				"exported_methods",
				"unexported_methods",
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

	// Validate type layout
	seen = make(map[string]bool)
	for _, elem := range c.Types.TypeLayout {
		if !ValidTypeLayoutElements[elem] {
			return fmt.Errorf("unknown type layout element: %q", elem)
		}
		if seen[elem] {
			return fmt.Errorf("duplicate type layout element: %q", elem)
		}
		seen[elem] = true
	}

	// Validate enum layout
	seen = make(map[string]bool)
	for _, elem := range c.Types.EnumLayout {
		if !ValidEnumLayoutElements[elem] {
			return fmt.Errorf("unknown enum layout element: %q", elem)
		}
		if seen[elem] {
			return fmt.Errorf("duplicate enum layout element: %q", elem)
		}
		seen[elem] = true
	}

	if !ValidModes[c.Behavior.Mode] {
		return fmt.Errorf("unknown mode: %q (valid: strict, warn, append, drop)", c.Behavior.Mode)
	}

	return nil
}

// ErrInvalidConfig is returned when config validation fails.
var ErrInvalidConfig = errors.New("invalid config")

// LoadConfig loads configuration from a TOML file.
// If the file doesn't exist, returns default config.
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	// Parse TOML into a separate struct to detect what was actually set
	var fileCfg fileConfig
	if _, err := toml.DecodeFile(path, &fileCfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Merge file config into defaults
	if fileCfg.Behavior.Mode != "" {
		cfg.Behavior.Mode = fileCfg.Behavior.Mode
	}
	if fileCfg.Sections.Order != nil {
		cfg.Sections.Order = fileCfg.Sections.Order
	}
	if fileCfg.Types.TypeLayout != nil {
		cfg.Types.TypeLayout = fileCfg.Types.TypeLayout
	}
	if fileCfg.Types.EnumLayout != nil {
		cfg.Types.EnumLayout = fileCfg.Types.EnumLayout
	}

	// Validate the merged config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// fileConfig mirrors Config but uses pointers/nil to detect unset values.
type fileConfig struct {
	Sections fileSectionsConfig
	Types    fileTypesConfig
	Behavior fileBehaviorConfig
}

type fileSectionsConfig struct {
	Order []string
}

type fileTypesConfig struct {
	TypeLayout []string `toml:"type_layout"`
	EnumLayout []string `toml:"enum_layout"`
}

type fileBehaviorConfig struct {
	Mode string
}