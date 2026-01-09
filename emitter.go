package reorder

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/toejough/go-reorder/internal/categorize"
)

// unexported variables.
var (
	emitters = map[string]sectionEmitter{
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
)

// sectionEmitter emits declarations for a section from categorized declarations.
type sectionEmitter func(*categorize.CategorizedDecls, *Config) []dst.Decl

// emitEnumGroup emits a single enum group using the specified layout.
func emitEnumGroup(eg *categorize.EnumGroup, layout []string) []dst.Decl {
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

// emitEnumGroups emits all enum groups using the specified layout.
func emitEnumGroups(groups []*categorize.EnumGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, enumGrp := range groups {
		decls = append(decls, emitEnumGroup(enumGrp, layout)...)
	}
	return decls
}

func emitExportedConsts(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.ExportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{categorize.MergeConstSpecs(cat.ExportedConsts, "Exported constants.")}
}

func emitExportedEnums(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return emitEnumGroups(cat.ExportedEnums, cfg.Types.EnumLayout)
}

func emitExportedFuncs(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return emitFuncs(cat.ExportedFuncs)
}

func emitExportedTypes(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return emitTypeGroups(cat.ExportedTypes, cfg.Types.TypeLayout)
}

func emitExportedVars(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.ExportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{categorize.MergeVarSpecs(cat.ExportedVars, "Exported variables.")}
}

// emitFuncs emits standalone functions with proper spacing.
func emitFuncs(funcs []*dst.FuncDecl) []dst.Decl {
	decls := make([]dst.Decl, 0, len(funcs))
	for _, fn := range funcs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}
	return decls
}

func emitImports(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if cat.Imports == nil {
		return []dst.Decl{}
	}
	return cat.Imports
}

func emitInit(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	decls := make([]dst.Decl, 0, len(cat.Init))
	for _, fn := range cat.Init {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}
	return decls
}

func emitMain(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if cat.Main == nil {
		return []dst.Decl{}
	}
	return []dst.Decl{cat.Main}
}

// emitTypeGroup emits a single type group using the specified layout.
func emitTypeGroup(tg *categorize.TypeGroup, layout []string) []dst.Decl {
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

// emitTypeGroups emits all type groups using the specified layout.
func emitTypeGroups(groups []*categorize.TypeGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, typeGrp := range groups {
		decls = append(decls, emitTypeGroup(typeGrp, layout)...)
	}
	return decls
}

func emitUncategorized(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if cat.Uncategorized == nil {
		return []dst.Decl{}
	}
	return cat.Uncategorized
}

func emitUnexportedConsts(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.UnexportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{categorize.MergeConstSpecs(cat.UnexportedConsts, "unexported constants.")}
}

func emitUnexportedEnums(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return emitEnumGroups(cat.UnexportedEnums, cfg.Types.EnumLayout)
}

func emitUnexportedFuncs(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	return emitFuncs(cat.UnexportedFuncs)
}

func emitUnexportedTypes(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
	return emitTypeGroups(cat.UnexportedTypes, cfg.Types.TypeLayout)
}

func emitUnexportedVars(cat *categorize.CategorizedDecls, _ *Config) []dst.Decl {
	if len(cat.UnexportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{categorize.MergeVarSpecs(cat.UnexportedVars, "unexported variables.")}
}

// getEmitter returns the emitter for a section name, or nil if unknown.
func getEmitter(section string) sectionEmitter {
	return emitters[section]
}
