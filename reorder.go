// Package reorder provides tools for reorganizing Go source code declarations.
//
// The package reorders declarations in Go files according to configurable conventions,
// grouping related code together (types with their methods and constructors) and
// organizing sections in a consistent order.
//
// # Basic Usage
//
// The simplest way to reorder code is with default settings:
//
//	result, err := reorder.Source(srcCode)
//
// # Custom Configuration
//
// For custom ordering, load or create a config:
//
//	cfg, err := reorder.LoadConfig(".go-reorder.toml")
//	result, err := reorder.SourceWithConfig(srcCode, cfg)
//
// Or modify the default config:
//
//	cfg := reorder.DefaultConfig()
//	cfg.Sections.Order = []string{"imports", "exported_types", "unexported_types"}
//	cfg.Behavior.Mode = "append"
//	result, err := reorder.SourceWithConfig(srcCode, cfg)
//
// # Section Ordering
//
// Declarations are organized into sections: imports, main, init, exported/unexported
// consts/enums/vars/types/funcs, and uncategorized. The order of these sections
// is configurable.
//
// # Type Grouping
//
// Types are automatically grouped with:
//   - Constructors: Functions named New*TypeName (e.g., NewUser, NewMockUser)
//   - Methods: Both exported and unexported methods on the type
//
// Enums (types with associated iota const blocks) are similarly grouped.
package reorder

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/go-reorder/internal/categorize"
	"github.com/toejough/go-reorder/internal/reassemble"
)

// StrictModeError is returned when mode is "strict" and code has no matching section.
type StrictModeError struct {
	ExcludedSections []string
}

// Error returns a helpful error message with hints for fixing the issue.
func (e *StrictModeError) Error() string {
	sections := strings.Join(e.ExcludedSections, ", ")
	return fmt.Sprintf(
		"strict mode: code has no matching section for: %s\n"+
			"Hints:\n"+
			"  - Add the missing section(s) to your config's [sections] order array\n"+
			"  - Add \"uncategorized\" to catch any unmatched code\n"+
			"  - Use --mode=append to be lenient (append unmatched code at end)\n"+
			"  - Use --mode=warn to append with a warning",
		sections,
	)
}

// Section represents a declaration section in a Go file.
type Section struct {
	Name     string // e.g., "Imports", "Exported Types", "unexported functions"
	Position int    // Position in file (1-indexed)
	Expected int    // Expected position (1-indexed), 0 if section shouldn't exist
}

// SectionOrder represents the detected sections in a file and their order.
type SectionOrder struct {
	Sections []Section
}

// AnalyzeSectionOrder analyzes the current declaration order in source code.
// Returns a SectionOrder showing which sections are present and their positions.
func AnalyzeSectionOrder(src string) (*SectionOrder, error) {
	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// Map section names to their expected positions (1-indexed)
	//nolint:mnd // These are the canonical ordering positions from CLAUDE.md
	expectedPositions := map[string]int{
		"Imports":              1,
		"main()":               2,
		"Exported Constants":   3,
		"Exported Enums":       4,
		"Exported Variables":   5,
		"Exported Types":       6,
		"Exported Functions":   7,
		"unexported constants": 8,
		"unexported enums":     9,
		"unexported variables": 10,
		"unexported types":     11,
		"unexported functions": 12,
	}

	// Track which sections we've seen and their first occurrence position
	sectionPositions := make(map[string]int)
	currentPos := 0

	// Walk through original declarations to track section transitions
	for _, decl := range file.Decls {
		currentPos++

		sectionName := categorize.IdentifySection(decl)
		if sectionName == "" {
			continue
		}

		// Record first occurrence of each section
		if _, seen := sectionPositions[sectionName]; !seen {
			sectionPositions[sectionName] = currentPos
		}
	}

	// Build the section list
	sections := make([]Section, 0, len(sectionPositions))
	for name, pos := range sectionPositions {
		sections = append(sections, Section{
			Name:     name,
			Position: pos,
			Expected: expectedPositions[name],
		})
	}

	// Sort by current position
	slices.SortFunc(sections, func(a, b Section) int {
		return a.Position - b.Position
	})

	return &SectionOrder{Sections: sections}, nil
}

// File reorders declarations in a dst.File according to project conventions.
func File(file *dst.File) error {
	cat := categorize.CategorizeDeclarations(file)
	reordered := reassemble.Declarations(cat)
	file.Decls = reordered

	return nil
}

// FileWithConfig reorders declarations in a dst.File using the provided configuration.
// Returns an error in strict mode if code has no matching section in the config.
func FileWithConfig(file *dst.File, cfg *Config) error {
	cat := categorize.CategorizeDeclarations(file)

	// Build section set for checking
	configSections := make(map[string]bool)
	for _, s := range cfg.Sections.Order {
		configSections[s] = true
	}

	// In strict mode, check for excluded sections before processing
	if cfg.Behavior.Mode == "strict" {
		excluded := categorize.FindExcludedSections(cat, configSections)
		if len(excluded) > 0 {
			return &StrictModeError{ExcludedSections: excluded}
		}
	}

	reassembleCfg := &reassemble.Config{
		Order:      cfg.Sections.Order,
		TypeLayout: cfg.Types.TypeLayout,
		EnumLayout: cfg.Types.EnumLayout,
		Mode:       cfg.Behavior.Mode,
	}

	reordered := reassemble.DeclarationsWithOrder(cat, reassembleCfg)
	file.Decls = reordered

	return nil
}

// Source reorders declarations in Go source code according to default conventions.
// It preserves all comments and handles edge cases like iota blocks and type-method grouping.
//
// Default ordering: imports, main, init, exported (consts, enums, vars, types, funcs),
// then unexported equivalents, then uncategorized.
//
// Types are grouped with their constructors (New*TypeName) and methods.
// Enums (iota types) are grouped with their const blocks.
//
// Example:
//
//	reordered, err := reorder.Source(srcCode)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(reordered)
func Source(src string) (string, error) {
	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		return "", fmt.Errorf("failed to parse source: %w", err)
	}

	err = File(file)
	if err != nil {
		return "", fmt.Errorf("failed to reorder: %w", err)
	}

	var buf bytes.Buffer

	res := decorator.NewRestorer()

	err = res.Fprint(&buf, file)
	if err != nil {
		return "", fmt.Errorf("failed to print: %w", err)
	}

	return buf.String(), nil
}

// SourceWithConfig reorders declarations using the provided configuration.
//
// Example with loaded config:
//
//	cfg, err := reorder.LoadConfig(".go-reorder.toml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := reorder.SourceWithConfig(src, cfg)
//
// Example with modified default config:
//
//	cfg := reorder.DefaultConfig()
//	cfg.Behavior.Mode = "append"  // Don't error on unmatched code
//	result, err := reorder.SourceWithConfig(src, cfg)
func SourceWithConfig(src string, cfg *Config) (string, error) {
	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		return "", fmt.Errorf("failed to parse source: %w", err)
	}

	err = FileWithConfig(file, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to reorder: %w", err)
	}

	var buf bytes.Buffer

	res := decorator.NewRestorer()

	err = res.Fprint(&buf, file)
	if err != nil {
		return "", fmt.Errorf("failed to print: %w", err)
	}

	return buf.String(), nil
}

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
