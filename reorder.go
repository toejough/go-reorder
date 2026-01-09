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

// Source reorders declarations in Go source code according to project conventions.
// It preserves all comments and handles edge cases like iota blocks and type-method grouping.
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
