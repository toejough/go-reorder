package reorder

import (
	"fmt"

	"github.com/dave/dst"
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
type sectionEmitter func(*categorizedDecls, *Config) []dst.Decl

// emitEnumGroup emits a single enum group using the specified layout.
func emitEnumGroup(eg *enumGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, elem := range layout {
		switch elem {
		case "typedef":
			if eg.typeDecl != nil {
				eg.typeDecl.Decs.Before = dst.EmptyLine
				decls = append(decls, eg.typeDecl)
			}
		case "iota":
			eg.constDecl.Decs.Start = nil
			eg.constDecl.Decs.Before = dst.EmptyLine
			eg.constDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", eg.typeName))
			decls = append(decls, eg.constDecl)
		case "exported_methods":
			for _, method := range eg.exportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		case "unexported_methods":
			for _, method := range eg.unexportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		}
	}
	return decls
}

// emitEnumGroups emits all enum groups using the specified layout.
func emitEnumGroups(groups []*enumGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, enumGrp := range groups {
		decls = append(decls, emitEnumGroup(enumGrp, layout)...)
	}
	return decls
}

func emitExportedConsts(cat *categorizedDecls, _ *Config) []dst.Decl {
	if len(cat.exportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeConstSpecs(cat.exportedConsts, "Exported constants.")}
}

func emitExportedEnums(cat *categorizedDecls, cfg *Config) []dst.Decl {
	return emitEnumGroups(cat.exportedEnums, cfg.Types.EnumLayout)
}

func emitExportedFuncs(cat *categorizedDecls, _ *Config) []dst.Decl {
	return emitFuncs(cat.exportedFuncs)
}

func emitExportedTypes(cat *categorizedDecls, cfg *Config) []dst.Decl {
	return emitTypeGroups(cat.exportedTypes, cfg.Types.TypeLayout)
}

func emitExportedVars(cat *categorizedDecls, _ *Config) []dst.Decl {
	if len(cat.exportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeVarSpecs(cat.exportedVars, "Exported variables.")}
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

func emitImports(cat *categorizedDecls, _ *Config) []dst.Decl {
	if cat.imports == nil {
		return []dst.Decl{}
	}
	return cat.imports
}

func emitInit(cat *categorizedDecls, _ *Config) []dst.Decl {
	decls := make([]dst.Decl, 0, len(cat.init))
	for _, fn := range cat.init {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}
	return decls
}

func emitMain(cat *categorizedDecls, _ *Config) []dst.Decl {
	if cat.main == nil {
		return []dst.Decl{}
	}
	return []dst.Decl{cat.main}
}

// emitTypeGroup emits a single type group using the specified layout.
func emitTypeGroup(tg *typeGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, elem := range layout {
		switch elem {
		case "typedef":
			if tg.typeDecl != nil {
				tg.typeDecl.Decs.Before = dst.EmptyLine
				decls = append(decls, tg.typeDecl)
			}
		case "constructors":
			for _, ctor := range tg.constructors {
				ctor.Decs.Before = dst.EmptyLine
				decls = append(decls, ctor)
			}
		case "exported_methods":
			for _, method := range tg.exportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		case "unexported_methods":
			for _, method := range tg.unexportedMethods {
				method.Decs.Before = dst.EmptyLine
				decls = append(decls, method)
			}
		}
	}
	return decls
}

// emitTypeGroups emits all type groups using the specified layout.
func emitTypeGroups(groups []*typeGroup, layout []string) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, typeGrp := range groups {
		decls = append(decls, emitTypeGroup(typeGrp, layout)...)
	}
	return decls
}

func emitUncategorized(cat *categorizedDecls, _ *Config) []dst.Decl {
	if cat.uncategorized == nil {
		return []dst.Decl{}
	}
	return cat.uncategorized
}

func emitUnexportedConsts(cat *categorizedDecls, _ *Config) []dst.Decl {
	if len(cat.unexportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeConstSpecs(cat.unexportedConsts, "unexported constants.")}
}

func emitUnexportedEnums(cat *categorizedDecls, cfg *Config) []dst.Decl {
	return emitEnumGroups(cat.unexportedEnums, cfg.Types.EnumLayout)
}

func emitUnexportedFuncs(cat *categorizedDecls, _ *Config) []dst.Decl {
	return emitFuncs(cat.unexportedFuncs)
}

func emitUnexportedTypes(cat *categorizedDecls, cfg *Config) []dst.Decl {
	return emitTypeGroups(cat.unexportedTypes, cfg.Types.TypeLayout)
}

func emitUnexportedVars(cat *categorizedDecls, _ *Config) []dst.Decl {
	if len(cat.unexportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeVarSpecs(cat.unexportedVars, "unexported variables.")}
}

// getEmitter returns the emitter for a section name, or nil if unknown.
func getEmitter(section string) sectionEmitter {
	return emitters[section]
}
