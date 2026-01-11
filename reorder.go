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
	"fmt"
	"go/token"
	"slices"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/go-reorder/internal/categorize"
	"github.com/toejough/go-reorder/internal/reassemble"
)

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
func FileWithConfig(file *dst.File, cfg *Config) error {
	cat := categorize.CategorizeDeclarations(file)

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
