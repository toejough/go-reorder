package reorder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Exported constants.
const (
	ConfigFileName = ".go-reorder.toml"
)

// Exported variables.
var (
	ErrInvalidConfig        = errors.New("invalid config")
	ValidEnumLayoutElements = map[string]bool{
		"typedef":            true,
		"iota":               true,
		"exported_methods":   true,
		"unexported_methods": true,
	}
	ValidModes = map[string]bool{
		"strict": true,
		"warn":   true,
		"append": true,
		"drop":   true,
	}
	ValidSections = map[string]bool{
		"imports":           true,
		"main":              true,
		"init":              true,
		"exported_consts":   true,
		"exported_enums":    true,
		"exported_vars":     true,
		"exported_types":    true,
		"exported_funcs":    true,
		"unexported_consts": true,
		"unexported_enums":  true,
		"unexported_vars":   true,
		"unexported_types":  true,
		"unexported_funcs":  true,
		"uncategorized":     true,
	}
	ValidTypeLayoutElements = map[string]bool{
		"typedef":            true,
		"constructors":       true,
		"exported_methods":   true,
		"unexported_methods": true,
	}
)

// BehaviorConfig controls how the reorderer handles edge cases.
//
// Mode determines what happens when declarations don't match any section in the config:
//   - "strict": Return an error (default). Safe for CI.
//   - "warn":   Append unmatched code at end and print warning to stderr.
//   - "append": Silently append unmatched code at end.
//   - "drop":   Discard unmatched code. Useful for extracting specific sections.
type BehaviorConfig struct {
	// Mode controls handling of unmatched declarations.
	// Valid values: "strict", "warn", "append", "drop".
	Mode string
}

// Config holds all configuration for go-reorder.
//
// Example usage:
//
//	cfg := reorder.DefaultConfig()
//	cfg.Behavior.Mode = "append"
//	result, err := reorder.SourceWithConfig(src, cfg)
//
// Or load from file:
//
//	cfg, err := reorder.LoadConfig(".go-reorder.toml")
//	result, err := reorder.SourceWithConfig(src, cfg)
type Config struct {
	// Sections controls the order of declaration groups in the output.
	Sections SectionsConfig

	// Types controls how types and enums are laid out within their sections.
	Types TypesConfig

	// Behavior controls error handling for unmatched declarations.
	Behavior BehaviorConfig
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

// SectionsConfig controls declaration ordering.
//
// Available section names:
//   - "imports":           Import declarations
//   - "main":              The main() function
//   - "init":              All init() functions (original order preserved)
//   - "exported_consts":   Exported constant declarations
//   - "exported_enums":    Exported enum types with their iota blocks
//   - "exported_vars":     Exported variable declarations
//   - "exported_types":    Exported type definitions (with constructors and methods)
//   - "exported_funcs":    Exported standalone functions
//   - "unexported_consts": Unexported constant declarations
//   - "unexported_enums":  Unexported enum types with their iota blocks
//   - "unexported_vars":   Unexported variable declarations
//   - "unexported_types":  Unexported type definitions (with constructors and methods)
//   - "unexported_funcs":  Unexported standalone functions
//   - "uncategorized":     Catch-all for anything not matching other sections
type SectionsConfig struct {
	// Order lists section names in the desired output order.
	// Sections not in this list will be handled according to Behavior.Mode.
	Order []string
}

// TypesConfig controls how types and enums are laid out internally.
//
// TypeLayout elements control type group ordering:
//   - "typedef":            The type definition itself (type Foo struct{})
//   - "constructors":       Functions matching New*TypeName (e.g., NewFoo, NewMockFoo)
//   - "exported_methods":   Exported methods on the type
//   - "unexported_methods": Unexported methods on the type
//
// EnumLayout elements control enum group ordering:
//   - "typedef":            The enum type definition (type Status int)
//   - "iota":               The associated iota const block
//   - "exported_methods":   Exported methods (e.g., String())
//   - "unexported_methods": Unexported methods
//
// Constructor matching: Functions are matched as constructors if they:
//   - Are named New + TypeName (e.g., NewUser for type User)
//   - Are named New + Prefix + TypeName (e.g., NewMockUser for type User)
//   - Return *TypeName or TypeName as first return value
type TypesConfig struct {
	// TypeLayout orders elements within each type group.
	TypeLayout []string

	// EnumLayout orders elements within each enum group.
	EnumLayout []string
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

// FindConfig searches for a config file starting from the given directory,
// walking up the directory tree until it finds one or reaches a boundary.
// Returns empty string if no config file is found.
// Boundaries are: .git directory, go.mod file, or filesystem root.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		// Check for config file in current directory
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Check for boundaries
		gitPath := filepath.Join(dir, ".git")
		goModPath := filepath.Join(dir, "go.mod")

		if _, err := os.Stat(gitPath); err == nil {
			// Found .git, stop here (don't go above)
			return "", nil
		}
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, stop here (don't go above)
			return "", nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

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

type fileBehaviorConfig struct {
	Mode string
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
