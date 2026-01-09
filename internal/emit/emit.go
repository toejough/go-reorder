// Package emit provides functions for emitting Go declarations in order.
package emit

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/toejough/go-reorder/internal/categorize"
)

// Config holds the layout configuration for emitting declarations.
type Config struct {
	TypeLayout []string
	EnumLayout []string
}

// SectionEmitter emits declarations for a section from categorized declarations.
type SectionEmitter func(*categorize.CategorizedDecls, *Config) []dst.Decl

// emitters maps section names to their emitter functions.
var emitters = map[string]SectionEmitter{
	"imports":           emitImports,
	"main":              emitMain,
	"init":              emitInit,
	"exported_consts":   emitExportedConsts,
	"exported_enums":    emitExportedEnums,
	"exported_vars":     emitExportedVars,
	"exported_types":    emitExportedTypes,
	"exported_funcs":    emitExportedFuncs,
	"unexported_consts": emitUnexportedConsts,
	"unexported_enums":  emitUnexportedEnums,
	"unexported_vars":   emitUnexportedVars,
	"unexported_types":  emitUnexportedTypes,
	"unexported_funcs":  emitUnexportedFuncs,
	"uncategorized":     emitUncategorized,
}

// GetEmitter returns the emitter for a section name, or nil if unknown.
func GetEmitter(section string) SectionEmitter {
	return emitters[section]
}

// EmitTypeGroup emits a single type group using the specified layout.
func EmitTypeGroup(tg *categorize.TypeGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)

	for _, elem := range layout {
		switch elem {
		case "typedef":
			if tg.TypeDecl != nil {
				tg.TypeDecl.Decs.Before = dst.EmptyLine
				decls = append(decls, tg.TypeDecl)
			}
		case "constructors":
			for _, ctor := range tg.Constructors {
				ctor.Decs.Before = dst.EmptyLine
				decls = append(decls, ctor)
			}
		case "exported_methods":
			for _, method := range tg.ExportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		case "unexported_methods":
			for _, method := range tg.UnexportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		}
	}

	return decls
}

// EmitTypeGroups emits all type groups using the specified layout.
func EmitTypeGroups(groups []*categorize.TypeGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)

	for _, typeGrp := range groups {
		decls = append(decls, EmitTypeGroup(typeGrp, layout)...)
	}

	return decls
}

// EmitEnumGroup emits a single enum group using the specified layout.
func EmitEnumGroup(eg *categorize.EnumGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)

	for _, elem := range layout {
		switch elem {
		case "typedef":
			if eg.TypeDecl != nil {
				eg.TypeDecl.Decs.Before = dst.EmptyLine
				decls = append(decls, eg.TypeDecl)
			}
		case "iota":
			eg.ConstDecl.Decs.Start = nil
			eg.ConstDecl.Decs.Before = dst.EmptyLine
			eg.ConstDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", eg.TypeName))
			decls = append(decls, eg.ConstDecl)
		case "exported_methods":
			for _, method := range eg.ExportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		case "unexported_methods":
			for _, method := range eg.UnexportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		}
	}

	return decls
}

// EmitEnumGroups emits all enum groups using the specified layout.
func EmitEnumGroups(groups []*categorize.EnumGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)

	for _, enumGrp := range groups {
		decls = append(decls, EmitEnumGroup(enumGrp, layout)...)
	}

	return decls
}

// EmitFuncs emits standalone functions with proper spacing.
func EmitFuncs(funcs []*dst.FuncDecl) []dst.Decl {
	decls := make([]dst.Decl, 0, len(funcs))

	for _, fn := range funcs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	return decls
}

// Imports returns the import declarations.
func Imports(cat *categorize.CategorizedDecls) []dst.Decl {
	if cat.Imports == nil {
		return []dst.Decl{}
	}

	return cat.Imports
}

// Main returns the main function declaration.
func Main(cat *categorize.CategorizedDecls) []dst.Decl {
	if cat.Main == nil {
		return []dst.Decl{}
	}

	return []dst.Decl{cat.Main}
}

// Init returns the init function declarations with proper spacing.
func Init(cat *categorize.CategorizedDecls) []dst.Decl {
	decls := make([]dst.Decl, 0, len(cat.Init))

	for _, fn := range cat.Init {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	return decls
}

// Section emitter implementations

func emitImports(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return Imports(cat)
}

func emitMain(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return Main(cat)
}

func emitInit(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return Init(cat)
}

func emitExportedConsts(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.ExportedConsts) == 0 {
		return []dst.Decl{}
	}

	return []dst.Decl{categorize.MergeConstSpecs(cat.ExportedConsts, "Exported constants.")}
}

func emitExportedEnums(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return EmitEnumGroups(cat.ExportedEnums, cfg.EnumLayout)
}

func emitExportedVars(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.ExportedVars) == 0 {
		return []dst.Decl{}
	}

	return []dst.Decl{categorize.MergeVarSpecs(cat.ExportedVars, "Exported variables.")}
}

func emitExportedTypes(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return EmitTypeGroups(cat.ExportedTypes, cfg.TypeLayout)
}

func emitExportedFuncs(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return EmitFuncs(cat.ExportedFuncs)
}

func emitUnexportedConsts(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.UnexportedConsts) == 0 {
		return []dst.Decl{}
	}

	return []dst.Decl{categorize.MergeConstSpecs(cat.UnexportedConsts, "unexported constants.")}
}

func emitUnexportedEnums(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return EmitEnumGroups(cat.UnexportedEnums, cfg.EnumLayout)
}

func emitUnexportedVars(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.UnexportedVars) == 0 {
		return []dst.Decl{}
	}

	return []dst.Decl{categorize.MergeVarSpecs(cat.UnexportedVars, "unexported variables.")}
}

func emitUnexportedTypes(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return EmitTypeGroups(cat.UnexportedTypes, cfg.TypeLayout)
}

func emitUnexportedFuncs(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return EmitFuncs(cat.UnexportedFuncs)
}

func emitUncategorized(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if cat.Uncategorized == nil {
		return []dst.Decl{}
	}

	return cat.Uncategorized
}
