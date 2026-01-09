// Package ast provides utility functions for working with Go AST nodes.
package ast

import (
	"go/token"
	"slices"
	"unicode"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

// IsExported returns true if the name starts with an uppercase letter.
func IsExported(name string) bool {
	if name == "" {
		return false
	}

	r := []rune(name)[0]

	return unicode.IsUpper(r)
}

// ExtractTypeName extracts the type name from a type expression.
func ExtractTypeName(expr dst.Expr) string {
	switch typeExpr := expr.(type) {
	case *dst.Ident:
		return typeExpr.Name
	case *dst.SelectorExpr:
		return typeExpr.Sel.Name
	case *dst.StarExpr:
		return ExtractTypeName(typeExpr.X)
	case *dst.IndexExpr:
		return ExtractTypeName(typeExpr.X)
	case *dst.IndexListExpr:
		return ExtractTypeName(typeExpr.X)
	}

	return ""
}

// ExtractReceiverTypeName extracts the type name from a method receiver.
func ExtractReceiverTypeName(recv *dst.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	return ExtractTypeName(recv.List[0].Type)
}

// ContainsIota checks if an expression contains the iota identifier.
func ContainsIota(expr dst.Expr) bool {
	if expr == nil {
		return false
	}

	found := false

	dstutil.Apply(expr, func(c *dstutil.Cursor) bool {
		if ident, ok := c.Node().(*dst.Ident); ok {
			if ident.Name == "iota" {
				found = true
				return false
			}
		}

		return true
	}, nil)

	return found
}

// IsIotaBlock checks if a const block uses iota.
func IsIotaBlock(decl *dst.GenDecl) bool {
	if decl.Tok != token.CONST {
		return false
	}

	for _, spec := range decl.Specs {
		vspec, ok := spec.(*dst.ValueSpec)
		if !ok {
			continue
		}

		if slices.ContainsFunc(vspec.Values, ContainsIota) {
			return true
		}
	}

	return false
}

// ExtractEnumType extracts the type name from an enum const block.
func ExtractEnumType(decl *dst.GenDecl) string {
	if len(decl.Specs) == 0 {
		return ""
	}

	vspec, ok := decl.Specs[0].(*dst.ValueSpec)
	if !ok || vspec.Type == nil {
		return ""
	}

	return ExtractTypeName(vspec.Type)
}
