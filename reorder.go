package reorder

import (
	"bytes"
	"fmt"
	"go/token"
	"slices"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/go-reorder/internal/categorize"
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
	reordered := reassembleDeclarations(cat)
	file.Decls = reordered

	return nil
}

// FileWithConfig reorders declarations in a dst.File using the provided configuration.
func FileWithConfig(file *dst.File, cfg *Config) error {
	cat := categorize.CategorizeDeclarations(file)
	reordered := reassembleDeclarationsWithConfig(cat, cfg)
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

// reassembleDeclarations builds the final ordered declaration list.
//
//nolint:gocognit,cyclop,funlen // Complex by design - assembles all declaration categories in correct order
func reassembleDeclarations(cat *categorize.CategorizedDecls) []dst.Decl {
	const extraCapacity = 10 // Extra capacity for main + merged const/var blocks

	// Pre-allocate with estimated capacity
	estimatedSize := len(cat.Imports) + len(cat.ExportedConsts) + len(cat.ExportedEnums) +
		len(cat.ExportedVars) + len(cat.ExportedTypes) + len(cat.ExportedFuncs) +
		len(cat.UnexportedConsts) + len(cat.UnexportedEnums) + len(cat.UnexportedVars) +
		len(cat.UnexportedTypes) + len(cat.UnexportedFuncs) + extraCapacity

	decls := make([]dst.Decl, 0, estimatedSize)

	// Imports
	decls = append(decls, cat.Imports...)

	// main() if present
	if cat.Main != nil {
		decls = append(decls, cat.Main)
	}

	// Exported constants (merged)
	if len(cat.ExportedConsts) > 0 {
		constDecl := categorize.MergeConstSpecs(cat.ExportedConsts, "Exported constants.")
		decls = append(decls, constDecl)
	}

	// Exported enums (type + const block pairs + methods)
	for _, enumGrp := range cat.ExportedEnums {
		if enumGrp.TypeDecl != nil {
			enumGrp.TypeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, enumGrp.TypeDecl)
		}
		// Add comment header (clear existing first to avoid duplicates)
		enumGrp.ConstDecl.Decs.Start = nil
		enumGrp.ConstDecl.Decs.Before = dst.EmptyLine
		enumGrp.ConstDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", enumGrp.TypeName))
		decls = append(decls, enumGrp.ConstDecl)

		// Add methods (exported first, then unexported)
		for _, method := range enumGrp.ExportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range enumGrp.UnexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Exported variables (merged)
	if len(cat.ExportedVars) > 0 {
		varDecl := categorize.MergeVarSpecs(cat.ExportedVars, "Exported variables.")
		decls = append(decls, varDecl)
	}

	// Exported types (with constructors and methods)
	for _, typeGrp := range cat.ExportedTypes {
		if typeGrp.TypeDecl != nil {
			typeGrp.TypeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, typeGrp.TypeDecl)
		}

		for _, ctor := range typeGrp.Constructors {
			ctor.Decs.Before = dst.EmptyLine
			decls = append(decls, ctor)
		}

		for _, method := range typeGrp.ExportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range typeGrp.UnexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Exported standalone functions
	for _, fn := range cat.ExportedFuncs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	// Unexported constants (merged)
	if len(cat.UnexportedConsts) > 0 {
		constDecl := categorize.MergeConstSpecs(cat.UnexportedConsts, "unexported constants.")
		decls = append(decls, constDecl)
	}

	// Unexported enums (type + const block pairs + methods)
	for _, enumGrp := range cat.UnexportedEnums {
		if enumGrp.TypeDecl != nil {
			enumGrp.TypeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, enumGrp.TypeDecl)
		}
		// Add comment header (clear existing first to avoid duplicates)
		enumGrp.ConstDecl.Decs.Start = nil
		enumGrp.ConstDecl.Decs.Before = dst.EmptyLine
		enumGrp.ConstDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", enumGrp.TypeName))
		decls = append(decls, enumGrp.ConstDecl)

		// Add methods (exported first, then unexported)
		for _, method := range enumGrp.ExportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range enumGrp.UnexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Unexported variables (merged)
	if len(cat.UnexportedVars) > 0 {
		varDecl := categorize.MergeVarSpecs(cat.UnexportedVars, "unexported variables.")
		decls = append(decls, varDecl)
	}

	// Unexported types (with constructors and methods)
	for _, typeGrp := range cat.UnexportedTypes {
		if typeGrp.TypeDecl != nil {
			typeGrp.TypeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, typeGrp.TypeDecl)
		}

		for _, ctor := range typeGrp.Constructors {
			ctor.Decs.Before = dst.EmptyLine
			decls = append(decls, ctor)
		}

		for _, method := range typeGrp.ExportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range typeGrp.UnexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Unexported standalone functions
	for _, fn := range cat.UnexportedFuncs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	return decls
}

// reassembleDeclarationsWithConfig builds the ordered declaration list using config.
func reassembleDeclarationsWithConfig(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	// Build set of sections in config
	configSections := make(map[string]bool)
	for _, s := range cfg.Sections.Order {
		configSections[s] = true
	}

	// Collect uncategorized from sections not in config (if mode allows)
	if cfg.Behavior.Mode != "drop" {
		categorize.CollectUncategorized(cat, configSections)
	}

	decls := make([]dst.Decl, 0)

	for _, section := range cfg.Sections.Order {
		emitter := getEmitter(section)
		if emitter != nil {
			decls = append(decls, emitter(cat, cfg)...)
		}
	}

	return decls
}
