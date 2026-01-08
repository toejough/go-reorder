package reorder

import (
	"fmt"

	"github.com/dave/dst"
)

// sectionEmitter emits declarations for a section from categorized declarations.
type sectionEmitter func(*categorizedDecls) []dst.Decl

// emitters maps section names to their emitter functions.
var emitters = map[string]sectionEmitter{
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

// getEmitter returns the emitter for a section name, or nil if unknown.
func getEmitter(section string) sectionEmitter {
	return emitters[section]
}

func emitImports(cat *categorizedDecls) []dst.Decl {
	if cat.imports == nil {
		return []dst.Decl{}
	}
	return cat.imports
}

func emitMain(cat *categorizedDecls) []dst.Decl {
	if cat.main == nil {
		return []dst.Decl{}
	}
	return []dst.Decl{cat.main}
}

func emitInit(cat *categorizedDecls) []dst.Decl {
	// TODO: Phase 6 will add init function handling
	return []dst.Decl{}
}

func emitExportedConsts(cat *categorizedDecls) []dst.Decl {
	if len(cat.exportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeConstSpecs(cat.exportedConsts, "Exported constants.")}
}

func emitExportedEnums(cat *categorizedDecls) []dst.Decl {
	return emitEnumGroups(cat.exportedEnums)
}

func emitExportedVars(cat *categorizedDecls) []dst.Decl {
	if len(cat.exportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeVarSpecs(cat.exportedVars, "Exported variables.")}
}

func emitExportedTypes(cat *categorizedDecls) []dst.Decl {
	return emitTypeGroups(cat.exportedTypes)
}

func emitExportedFuncs(cat *categorizedDecls) []dst.Decl {
	return emitFuncs(cat.exportedFuncs)
}

func emitUnexportedConsts(cat *categorizedDecls) []dst.Decl {
	if len(cat.unexportedConsts) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeConstSpecs(cat.unexportedConsts, "unexported constants.")}
}

func emitUnexportedEnums(cat *categorizedDecls) []dst.Decl {
	return emitEnumGroups(cat.unexportedEnums)
}

func emitUnexportedVars(cat *categorizedDecls) []dst.Decl {
	if len(cat.unexportedVars) == 0 {
		return []dst.Decl{}
	}
	return []dst.Decl{mergeVarSpecs(cat.unexportedVars, "unexported variables.")}
}

func emitUnexportedTypes(cat *categorizedDecls) []dst.Decl {
	return emitTypeGroups(cat.unexportedTypes)
}

func emitUnexportedFuncs(cat *categorizedDecls) []dst.Decl {
	return emitFuncs(cat.unexportedFuncs)
}

func emitUncategorized(cat *categorizedDecls) []dst.Decl {
	// TODO: Phase 6 will add uncategorized handling
	return []dst.Decl{}
}

// emitEnumGroups emits all enum groups (type + const + methods).
func emitEnumGroups(groups []*enumGroup) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, enumGrp := range groups {
		if enumGrp.typeDecl != nil {
			enumGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, enumGrp.typeDecl)
		}
		enumGrp.constDecl.Decs.Start = nil
		enumGrp.constDecl.Decs.Before = dst.EmptyLine
		enumGrp.constDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", enumGrp.typeName))
		decls = append(decls, enumGrp.constDecl)

		for _, method := range enumGrp.exportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
		for _, method := range enumGrp.unexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}
	return decls
}

// emitTypeGroups emits all type groups (type + constructors + methods).
func emitTypeGroups(groups []*typeGroup) []dst.Decl {
	decls := make([]dst.Decl, 0)
	for _, typeGrp := range groups {
		if typeGrp.typeDecl != nil {
			typeGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, typeGrp.typeDecl)
		}
		for _, ctor := range typeGrp.constructors {
			ctor.Decs.Before = dst.EmptyLine
			decls = append(decls, ctor)
		}
		for _, method := range typeGrp.exportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
		for _, method := range typeGrp.unexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}
	return decls
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