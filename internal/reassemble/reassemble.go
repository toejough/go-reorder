// Package reassemble provides functions for reassembling Go declarations in order.
package reassemble

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/toejough/go-reorder/internal/categorize"
	"github.com/toejough/go-reorder/internal/emit"
)

// Config holds configuration for reassembly.
type Config struct {
	Order      []string // Section order
	TypeLayout []string // Layout for type groups
	EnumLayout []string // Layout for enum groups
	Mode       string   // Behavior mode: "preserve" or "drop"
}

// DefaultConfig returns the default reassembly configuration.
func DefaultConfig() *Config {
	return &Config{
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
		TypeLayout: []string{"typedef", "constructors", "exported_methods", "unexported_methods"},
		EnumLayout: []string{"typedef", "iota", "exported_methods", "unexported_methods"},
		Mode:       "preserve",
	}
}

// Declarations builds the final ordered declaration list using default order.
//
//nolint:gocognit,cyclop,funlen // Complex by design - assembles all declaration categories in correct order
func Declarations(cat *categorize.CategorizedDecls) []dst.Decl {
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

// DeclarationsWithOrder builds the ordered declaration list using config.
func DeclarationsWithOrder(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	// Build set of sections in config
	configSections := make(map[string]bool)
	for _, s := range cfg.Order {
		configSections[s] = true
	}

	// Collect uncategorized from sections not in config (if mode allows)
	if cfg.Mode != "drop" {
		categorize.CollectUncategorized(cat, configSections)
	}

	decls := make([]dst.Decl, 0)

	emitCfg := &emit.Config{
		TypeLayout: cfg.TypeLayout,
		EnumLayout: cfg.EnumLayout,
	}

	for _, section := range cfg.Order {
		emitter := emit.GetEmitter(section)
		if emitter != nil {
			decls = append(decls, emitter(cat, emitCfg)...)
		}
	}

	return decls
}
