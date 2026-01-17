// Package categorize provides declaration categorization for Go source files.
package categorize

import (
	"go/token"
	"sort"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/go-reorder/internal/ast"
)

// CategorizedDecls holds declarations organized by category.
type CategorizedDecls struct {
	Imports          []dst.Decl
	Main             *dst.FuncDecl
	Init             []*dst.FuncDecl
	ExportedConsts   []*dst.ValueSpec
	ExportedEnums    []*EnumGroup
	ExportedVars     []*dst.ValueSpec
	ExportedTypes    []*TypeGroup
	ExportedFuncs    []*dst.FuncDecl
	UnexportedConsts []*dst.ValueSpec
	UnexportedEnums  []*EnumGroup
	UnexportedVars   []*dst.ValueSpec
	UnexportedTypes  []*TypeGroup
	UnexportedFuncs  []*dst.FuncDecl
	Uncategorized    []dst.Decl
}

// EnumGroup pairs an enum type with its iota const block and associated methods.
type EnumGroup struct {
	TypeName          string
	TypeDecl          *dst.GenDecl
	ConstDecl         *dst.GenDecl
	ExportedMethods   []*dst.FuncDecl
	UnexportedMethods []*dst.FuncDecl
}

// TypeGroup holds a type and its associated constructors and methods.
type TypeGroup struct {
	TypeName          string
	TypeDecl          *dst.GenDecl
	Constructors      []*dst.FuncDecl
	ExportedMethods   []*dst.FuncDecl
	UnexportedMethods []*dst.FuncDecl
}

// getFirstReturnTypeName extracts the type name from the first return value of a function.
// Returns empty string if the function has no return values or the type can't be determined.
// Handles both direct types (TypeName) and pointer types (*TypeName).
func getFirstReturnTypeName(fn *dst.FuncDecl) string {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}

	firstResult := fn.Type.Results.List[0].Type

	// Handle pointer types: *TypeName
	if starExpr, ok := firstResult.(*dst.StarExpr); ok {
		if ident, ok := starExpr.X.(*dst.Ident); ok {
			return ident.Name
		}
		return ""
	}

	// Handle direct types: TypeName
	if ident, ok := firstResult.(*dst.Ident); ok {
		return ident.Name
	}

	return ""
}

// CategorizeDeclarations organizes all declarations by category.
//
// The algorithm uses four passes to properly handle Go's declaration patterns:
//
// Pass 1 - Collect type names: Builds a map of all type names defined in the file.
// This is needed before Pass 2 so we can identify constructors (NewTypeName patterns)
// and associate methods with their receiver types.
//
// Pass 2 - Categorize declarations: The main categorization pass. Processes each
// declaration and assigns it to the appropriate category (imports, consts, vars,
// types, funcs). Methods are associated with their receiver types, and constructors
// are matched to types using a longest-suffix match algorithm.
//
// Pass 3 - Pair enums with types: Enum const blocks (iota patterns) are identified
// in Pass 2, but their type definitions may appear separately. This pass pairs enum
// const blocks with their type definitions and transfers methods from TypeGroup to
// EnumGroup, then removes the type from the regular types list.
//
// Pass 4 - Add method-only types: Handles files that contain methods for types
// defined elsewhere. TypeGroups that have methods but no TypeDecl are added to
// the appropriate (exported/unexported) types list so they're not lost.
//
//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // Complex by nature - handles all Go declaration types
func CategorizeDeclarations(file *dst.File) *CategorizedDecls {
	cat := &CategorizedDecls{}

	// Maps for grouping types with their methods and constructors
	typeGroups := make(map[string]*TypeGroup)
	// Track which type names are enums (have iota const blocks)
	enumTypes := make(map[string]bool)

	// Pass 1: Collect all type names
	// We need to know all types before categorizing so we can:
	// - Match constructors (NewFoo) to their types (Foo)
	// - Associate methods with their receiver types
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if tspec, ok := spec.(*dst.TypeSpec); ok {
					typeName := tspec.Name.Name
					typeGroups[typeName] = &TypeGroup{TypeName: typeName}
				}
			}
		}
	}

	// Pass 2: Categorize all declarations
	for _, decl := range file.Decls {
		switch genDecl := decl.(type) {
		case *dst.GenDecl:
			//nolint:exhaustive // We only care about IMPORT/CONST/VAR/TYPE; other tokens are intentionally ignored
			switch genDecl.Tok {
			case token.IMPORT:
				cat.Imports = append(cat.Imports, genDecl)
			case token.CONST:
				// Check if this is a typed iota block (enum pattern)
				typeName := ""
				if ast.IsIotaBlock(genDecl) {
					typeName = ast.ExtractEnumType(genDecl)
				}

				if typeName != "" { //nolint:nestif // Categorization logic requires nested conditions
					// Typed iota block = enum
					exported := ast.IsExported(typeName)

					if exported {
						cat.ExportedEnums = append(cat.ExportedEnums, &EnumGroup{
							TypeName:  typeName,
							ConstDecl: genDecl,
						})
					} else {
						cat.UnexportedEnums = append(cat.UnexportedEnums, &EnumGroup{
							TypeName:  typeName,
							ConstDecl: genDecl,
						})
					}

					enumTypes[typeName] = true
				} else {
					// Regular const - extract specs for merging
					for _, spec := range genDecl.Specs {
						if vspec, ok := spec.(*dst.ValueSpec); ok {
							if len(vspec.Names) > 0 {
								exported := ast.IsExported(vspec.Names[0].Name)
								if exported {
									cat.ExportedConsts = append(cat.ExportedConsts, vspec)
								} else {
									cat.UnexportedConsts = append(cat.UnexportedConsts, vspec)
								}
							}
						}
					}
				}
			case token.VAR:
				// Extract specs for merging
				for _, spec := range genDecl.Specs {
					if vspec, ok := spec.(*dst.ValueSpec); ok {
						if len(vspec.Names) > 0 {
							exported := ast.IsExported(vspec.Names[0].Name)
							if exported {
								cat.ExportedVars = append(cat.ExportedVars, vspec)
							} else {
								cat.UnexportedVars = append(cat.UnexportedVars, vspec)
							}
						}
					}
				}
			case token.TYPE:
				// Extract type name
				for _, spec := range genDecl.Specs {
					if tspec, ok := spec.(*dst.TypeSpec); ok { //nolint:nestif // Type extraction requires nested type assertions
						typeName := tspec.Name.Name
						exported := ast.IsExported(typeName)

						// Create or get type group
						if typeGroups[typeName] == nil {
							typeGroups[typeName] = &TypeGroup{
								TypeName: typeName,
							}
						}

						// Create individual GenDecl for this type to avoid duplicate
						// node issues when a grouped type declaration is split across
						// multiple TypeGroups.
						typeGroups[typeName].TypeDecl = &dst.GenDecl{
							Tok:   token.TYPE,
							Specs: []dst.Spec{tspec},
						}

						// Add to categorized list if not an enum type
						if !enumTypes[typeName] {
							if exported {
								cat.ExportedTypes = append(cat.ExportedTypes, typeGroups[typeName])
							} else {
								cat.UnexportedTypes = append(cat.UnexportedTypes, typeGroups[typeName])
							}
						}
					}
				}
			default:
				// Other token types are ignored
			}
		case *dst.FuncDecl:
			switch {
			case genDecl.Name.Name == "main" && genDecl.Recv == nil:
				cat.Main = genDecl
			case genDecl.Name.Name == "init" && genDecl.Recv == nil:
				cat.Init = append(cat.Init, genDecl)
			case genDecl.Recv != nil:
				// Method - associate with type
				typeName := ast.ExtractReceiverTypeName(genDecl.Recv)
				if typeGroups[typeName] == nil {
					typeGroups[typeName] = &TypeGroup{
						TypeName: typeName,
					}
				}

				methodExported := ast.IsExported(genDecl.Name.Name)
				if methodExported {
					typeGroups[typeName].ExportedMethods = append(typeGroups[typeName].ExportedMethods, genDecl)
				} else {
					typeGroups[typeName].UnexportedMethods = append(typeGroups[typeName].UnexportedMethods, genDecl)
				}
			default:
				// Standalone function or constructor
				funcName := genDecl.Name.Name
				exported := ast.IsExported(funcName)

				// Check if it's a constructor (New* or Must* prefix, matched by return type)
				// Constructor matching algorithm (aligned with funcorder):
				// 1. Function must have New* or Must* prefix
				// 2. Match by return type: first return must be TypeName or *TypeName defined in this file
				isConstructorCandidate := strings.HasPrefix(funcName, "New") || strings.HasPrefix(funcName, "Must")
				if isConstructorCandidate {
					if returnType := getFirstReturnTypeName(genDecl); returnType != "" {
						if typeGroups[returnType] != nil {
							typeGroups[returnType].Constructors = append(typeGroups[returnType].Constructors, genDecl)
							continue
						}
					}
				}

				// Not a constructor, add to standalone functions
				if exported {
					cat.ExportedFuncs = append(cat.ExportedFuncs, genDecl)
				} else {
					cat.UnexportedFuncs = append(cat.UnexportedFuncs, genDecl)
				}
			}
		}
	}

	// Pass 3: Pair enum types with their const blocks
	// Enum const blocks were added to cat.ExportedEnums/UnexportedEnums in Pass 2,
	// but the corresponding type definitions went into typeGroups. This pass:
	// - Links the TypeDecl to each EnumGroup
	// - Transfers methods from TypeGroup to EnumGroup
	// - Removes the type from cat.ExportedTypes/UnexportedTypes (since it's now in Enums)
	for _, enumGroup := range cat.ExportedEnums {
		if typeGroups[enumGroup.TypeName] != nil {
			enumGroup.TypeDecl = typeGroups[enumGroup.TypeName].TypeDecl
			// Transfer methods from TypeGroup to EnumGroup
			enumGroup.ExportedMethods = typeGroups[enumGroup.TypeName].ExportedMethods
			enumGroup.UnexportedMethods = typeGroups[enumGroup.TypeName].UnexportedMethods
			// Remove from regular types
			for i, tg := range cat.ExportedTypes {
				if tg.TypeName == enumGroup.TypeName {
					cat.ExportedTypes = append(cat.ExportedTypes[:i], cat.ExportedTypes[i+1:]...)
					break
				}
			}
		}
	}

	for _, enumGroup := range cat.UnexportedEnums {
		if typeGroups[enumGroup.TypeName] != nil {
			enumGroup.TypeDecl = typeGroups[enumGroup.TypeName].TypeDecl
			// Transfer methods from TypeGroup to EnumGroup
			enumGroup.ExportedMethods = typeGroups[enumGroup.TypeName].ExportedMethods
			enumGroup.UnexportedMethods = typeGroups[enumGroup.TypeName].UnexportedMethods
			// Remove from regular types
			for i, tg := range cat.UnexportedTypes {
				if tg.TypeName == enumGroup.TypeName {
					cat.UnexportedTypes = append(cat.UnexportedTypes[:i], cat.UnexportedTypes[i+1:]...)
					break
				}
			}
		}
	}

	// Pass 4: Add method-only TypeGroups
	// Some files contain only methods for types defined elsewhere (common pattern for
	// test helpers, large types split across files, etc.). These TypeGroups have methods
	// but no TypeDecl (nil). We add them to the appropriate types list so they're included
	// in the output. Without this pass, methods in such files would be silently dropped.
	for _, tg := range typeGroups {
		if tg.TypeDecl == nil && (len(tg.ExportedMethods) > 0 || len(tg.UnexportedMethods) > 0) {
			if ast.IsExported(tg.TypeName) {
				cat.ExportedTypes = append(cat.ExportedTypes, tg)
			} else {
				cat.UnexportedTypes = append(cat.UnexportedTypes, tg)
			}
		}
	}

	// Sort everything
	SortCategorized(cat)

	return cat
}

// IdentifySection determines which section a declaration belongs to.
//
//nolint:gocognit,cyclop,funlen,nestif,varnamelen // Complex type checking is inherent to declaration categorization
func IdentifySection(decl dst.Decl) string {
	switch d := decl.(type) {
	case *dst.GenDecl:
		if d.Tok == token.IMPORT {
			return "Imports"
		}

		if d.Tok == token.CONST {
			// Only treat as enum if it's a typed iota block
			if ast.IsIotaBlock(d) {
				typeName := ast.ExtractEnumType(d)
				if typeName != "" {
					if ast.IsExported(typeName) {
						return "Exported Enums"
					}

					return "unexported enums"
				}
			}
			// Check if it's a merged const block
			if len(d.Specs) > 0 {
				if vspec, ok := d.Specs[0].(*dst.ValueSpec); ok {
					if len(vspec.Names) > 0 {
						if ast.IsExported(vspec.Names[0].Name) {
							return "Exported Constants"
						}

						return "unexported constants"
					}
				}
			}
		}

		if d.Tok == token.VAR {
			if len(d.Specs) > 0 {
				if vspec, ok := d.Specs[0].(*dst.ValueSpec); ok {
					if len(vspec.Names) > 0 {
						if ast.IsExported(vspec.Names[0].Name) {
							return "Exported Variables"
						}

						return "unexported variables"
					}
				}
			}
		}

		if d.Tok == token.TYPE {
			if len(d.Specs) > 0 {
				if tspec, ok := d.Specs[0].(*dst.TypeSpec); ok {
					if ast.IsExported(tspec.Name.Name) {
						return "Exported Types"
					}

					return "unexported types"
				}
			}
		}
	case *dst.FuncDecl:
		if d.Name.Name == "main" && d.Recv == nil {
			return "main()"
		}
		// Skip methods (they're part of type groups)
		if d.Recv != nil {
			typeName := ast.ExtractReceiverTypeName(d.Recv)
			if ast.IsExported(typeName) {
				return "Exported Types"
			}

			return "unexported types"
		}

		if ast.IsExported(d.Name.Name) {
			return "Exported Functions"
		}

		return "unexported functions"
	}

	return ""
}

// SortCategorized sorts all categorized declarations alphabetically.
func SortCategorized(cat *CategorizedDecls) {
	// Sort const specs by name
	sort.Slice(cat.ExportedConsts, func(i, j int) bool {
		return cat.ExportedConsts[i].Names[0].Name < cat.ExportedConsts[j].Names[0].Name
	})
	sort.Slice(cat.UnexportedConsts, func(i, j int) bool {
		return cat.UnexportedConsts[i].Names[0].Name < cat.UnexportedConsts[j].Names[0].Name
	})

	// Sort var specs by name
	sort.Slice(cat.ExportedVars, func(i, j int) bool {
		return cat.ExportedVars[i].Names[0].Name < cat.ExportedVars[j].Names[0].Name
	})
	sort.Slice(cat.UnexportedVars, func(i, j int) bool {
		return cat.UnexportedVars[i].Names[0].Name < cat.UnexportedVars[j].Names[0].Name
	})

	// Sort enum groups by type name and their methods
	sort.Slice(cat.ExportedEnums, func(i, j int) bool {
		return cat.ExportedEnums[i].TypeName < cat.ExportedEnums[j].TypeName
	})
	for _, enumGrp := range cat.ExportedEnums {
		sort.Slice(enumGrp.ExportedMethods, func(i, j int) bool {
			return enumGrp.ExportedMethods[i].Name.Name < enumGrp.ExportedMethods[j].Name.Name
		})
		sort.Slice(enumGrp.UnexportedMethods, func(i, j int) bool {
			return enumGrp.UnexportedMethods[i].Name.Name < enumGrp.UnexportedMethods[j].Name.Name
		})
	}

	sort.Slice(cat.UnexportedEnums, func(i, j int) bool {
		return cat.UnexportedEnums[i].TypeName < cat.UnexportedEnums[j].TypeName
	})
	for _, enumGrp := range cat.UnexportedEnums {
		sort.Slice(enumGrp.ExportedMethods, func(i, j int) bool {
			return enumGrp.ExportedMethods[i].Name.Name < enumGrp.ExportedMethods[j].Name.Name
		})
		sort.Slice(enumGrp.UnexportedMethods, func(i, j int) bool {
			return enumGrp.UnexportedMethods[i].Name.Name < enumGrp.UnexportedMethods[j].Name.Name
		})
	}

	// Sort type groups by type name
	sort.Slice(cat.ExportedTypes, func(i, j int) bool {
		return cat.ExportedTypes[i].TypeName < cat.ExportedTypes[j].TypeName
	})
	sort.Slice(cat.UnexportedTypes, func(i, j int) bool {
		return cat.UnexportedTypes[i].TypeName < cat.UnexportedTypes[j].TypeName
	})

	// Sort within each type group
	for _, typeGrp := range cat.ExportedTypes {
		sort.Slice(typeGrp.Constructors, func(i, j int) bool {
			return typeGrp.Constructors[i].Name.Name < typeGrp.Constructors[j].Name.Name
		})
		sort.Slice(typeGrp.ExportedMethods, func(i, j int) bool {
			return typeGrp.ExportedMethods[i].Name.Name < typeGrp.ExportedMethods[j].Name.Name
		})
		sort.Slice(typeGrp.UnexportedMethods, func(i, j int) bool {
			return typeGrp.UnexportedMethods[i].Name.Name < typeGrp.UnexportedMethods[j].Name.Name
		})
	}

	for _, typeGrp := range cat.UnexportedTypes {
		sort.Slice(typeGrp.Constructors, func(i, j int) bool {
			return typeGrp.Constructors[i].Name.Name < typeGrp.Constructors[j].Name.Name
		})
		sort.Slice(typeGrp.ExportedMethods, func(i, j int) bool {
			return typeGrp.ExportedMethods[i].Name.Name < typeGrp.ExportedMethods[j].Name.Name
		})
		sort.Slice(typeGrp.UnexportedMethods, func(i, j int) bool {
			return typeGrp.UnexportedMethods[i].Name.Name < typeGrp.UnexportedMethods[j].Name.Name
		})
	}

	// Sort standalone functions by name
	sort.Slice(cat.ExportedFuncs, func(i, j int) bool {
		return cat.ExportedFuncs[i].Name.Name < cat.ExportedFuncs[j].Name.Name
	})
	sort.Slice(cat.UnexportedFuncs, func(i, j int) bool {
		return cat.UnexportedFuncs[i].Name.Name < cat.UnexportedFuncs[j].Name.Name
	})
}

// CollectUncategorized moves declarations from excluded sections to uncategorized.
//
//nolint:funlen,gocognit,cyclop // Section handling is inherently repetitive
func CollectUncategorized(cat *CategorizedDecls, includedSections map[string]bool) {
	if !includedSections["exported_consts"] && len(cat.ExportedConsts) > 0 {
		cat.Uncategorized = append(cat.Uncategorized, MergeConstSpecs(cat.ExportedConsts, "Exported constants."))
		cat.ExportedConsts = nil
	}
	if !includedSections["exported_vars"] && len(cat.ExportedVars) > 0 {
		cat.Uncategorized = append(cat.Uncategorized, MergeVarSpecs(cat.ExportedVars, "Exported variables."))
		cat.ExportedVars = nil
	}
	if !includedSections["exported_funcs"] {
		for _, fn := range cat.ExportedFuncs {
			fn.Decs.Before = dst.EmptyLine
			cat.Uncategorized = append(cat.Uncategorized, fn)
		}
		cat.ExportedFuncs = nil
	}
	if !includedSections["unexported_consts"] && len(cat.UnexportedConsts) > 0 {
		cat.Uncategorized = append(cat.Uncategorized, MergeConstSpecs(cat.UnexportedConsts, "unexported constants."))
		cat.UnexportedConsts = nil
	}
	if !includedSections["unexported_vars"] && len(cat.UnexportedVars) > 0 {
		cat.Uncategorized = append(cat.Uncategorized, MergeVarSpecs(cat.UnexportedVars, "unexported variables."))
		cat.UnexportedVars = nil
	}
	if !includedSections["unexported_funcs"] {
		for _, fn := range cat.UnexportedFuncs {
			fn.Decs.Before = dst.EmptyLine
			cat.Uncategorized = append(cat.Uncategorized, fn)
		}
		cat.UnexportedFuncs = nil
	}
	// Handle types (includes type decl, constructors, methods)
	if !includedSections["exported_types"] {
		for _, tg := range cat.ExportedTypes {
			if tg.TypeDecl != nil {
				tg.TypeDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, tg.TypeDecl)
			}
			for _, ctor := range tg.Constructors {
				ctor.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, ctor)
			}
			for _, m := range tg.ExportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
			for _, m := range tg.UnexportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
		}
		cat.ExportedTypes = nil
	}
	if !includedSections["unexported_types"] {
		for _, tg := range cat.UnexportedTypes {
			if tg.TypeDecl != nil {
				tg.TypeDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, tg.TypeDecl)
			}
			for _, ctor := range tg.Constructors {
				ctor.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, ctor)
			}
			for _, m := range tg.ExportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
			for _, m := range tg.UnexportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
		}
		cat.UnexportedTypes = nil
	}
	// Handle enums (includes type decl, iota const, methods)
	if !includedSections["exported_enums"] {
		for _, eg := range cat.ExportedEnums {
			if eg.TypeDecl != nil {
				eg.TypeDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, eg.TypeDecl)
			}
			if eg.ConstDecl != nil {
				eg.ConstDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, eg.ConstDecl)
			}
			for _, m := range eg.ExportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
			for _, m := range eg.UnexportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
		}
		cat.ExportedEnums = nil
	}
	if !includedSections["unexported_enums"] {
		for _, eg := range cat.UnexportedEnums {
			if eg.TypeDecl != nil {
				eg.TypeDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, eg.TypeDecl)
			}
			if eg.ConstDecl != nil {
				eg.ConstDecl.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, eg.ConstDecl)
			}
			for _, m := range eg.ExportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
			for _, m := range eg.UnexportedMethods {
				m.Decs.Before = dst.EmptyLine
				cat.Uncategorized = append(cat.Uncategorized, m)
			}
		}
		cat.UnexportedEnums = nil
	}
}

// FindExcludedSections returns the names of sections that have content
// but are not in the includedSections map.
func FindExcludedSections(cat *CategorizedDecls, includedSections map[string]bool) []string {
	var excluded []string

	if !includedSections["main"] && cat.Main != nil {
		excluded = append(excluded, "main")
	}
	if !includedSections["init"] && len(cat.Init) > 0 {
		excluded = append(excluded, "init")
	}
	if !includedSections["exported_consts"] && len(cat.ExportedConsts) > 0 {
		excluded = append(excluded, "exported_consts")
	}
	if !includedSections["exported_vars"] && len(cat.ExportedVars) > 0 {
		excluded = append(excluded, "exported_vars")
	}
	if !includedSections["exported_funcs"] && len(cat.ExportedFuncs) > 0 {
		excluded = append(excluded, "exported_funcs")
	}
	if !includedSections["exported_types"] && len(cat.ExportedTypes) > 0 {
		excluded = append(excluded, "exported_types")
	}
	if !includedSections["exported_enums"] && len(cat.ExportedEnums) > 0 {
		excluded = append(excluded, "exported_enums")
	}
	if !includedSections["unexported_consts"] && len(cat.UnexportedConsts) > 0 {
		excluded = append(excluded, "unexported_consts")
	}
	if !includedSections["unexported_vars"] && len(cat.UnexportedVars) > 0 {
		excluded = append(excluded, "unexported_vars")
	}
	if !includedSections["unexported_funcs"] && len(cat.UnexportedFuncs) > 0 {
		excluded = append(excluded, "unexported_funcs")
	}
	if !includedSections["unexported_types"] && len(cat.UnexportedTypes) > 0 {
		excluded = append(excluded, "unexported_types")
	}
	if !includedSections["unexported_enums"] && len(cat.UnexportedEnums) > 0 {
		excluded = append(excluded, "unexported_enums")
	}
	if !includedSections["uncategorized"] && len(cat.Uncategorized) > 0 {
		excluded = append(excluded, "uncategorized")
	}

	return excluded
}

// newGenDeclTemplate creates a GenDecl by parsing a template to ensure proper
// internal state for DST's restorer. This is necessary because directly
// constructing a GenDecl struct loses internal tracking that DST uses for
// correct trailing comment placement on the last spec.
func newGenDeclTemplate(tok token.Token) *dst.GenDecl {
	var keyword string
	if tok == token.CONST {
		keyword = "const"
	} else {
		keyword = "var"
	}

	src := "package p\n" + keyword + " (\n\tx int\n)\n"
	dec := decorator.NewDecorator(token.NewFileSet())
	file, _ := dec.Parse(src)

	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == tok {
			return dst.Clone(genDecl).(*dst.GenDecl)
		}
	}

	// Fallback (should not happen)
	return &dst.GenDecl{Tok: tok, Lparen: true}
}

// MergeConstSpecs creates a single const block from multiple specs.
func MergeConstSpecs(specs []*dst.ValueSpec, comment string) *dst.GenDecl {
	dstSpecs := make([]dst.Spec, 0, len(specs))

	for _, spec := range specs {
		spec.Decs.Before = dst.NewLine
		spec.Decs.After = dst.NewLine
		dstSpecs = append(dstSpecs, spec)
	}

	decl := newGenDeclTemplate(token.CONST)
	decl.Specs = dstSpecs
	decl.Decs.Before = dst.EmptyLine
	decl.Decs.Start = nil
	decl.Decs.Start.Append("// " + comment)

	return decl
}

// MergeVarSpecs creates a single var block from multiple specs.
func MergeVarSpecs(specs []*dst.ValueSpec, comment string) *dst.GenDecl {
	dstSpecs := make([]dst.Spec, 0, len(specs))

	for _, spec := range specs {
		spec.Decs.Before = dst.NewLine
		spec.Decs.After = dst.NewLine
		dstSpecs = append(dstSpecs, spec)
	}

	decl := newGenDeclTemplate(token.VAR)
	decl.Specs = dstSpecs
	decl.Decs.Before = dst.EmptyLine
	decl.Decs.Start = nil
	decl.Decs.Start.Append("// " + comment)

	return decl
}
