package ast

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
)

func TestIsExported(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"uppercase", "Foo", true},
		{"lowercase", "foo", false},
		{"empty", "", false},
		{"underscore", "_foo", false},
		{"uppercase underscore", "_Foo", false},
		{"single upper", "F", true},
		{"single lower", "f", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExported(tt.input); got != tt.expected {
				t.Errorf("IsExported(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractTypeName(t *testing.T) {
	tests := []struct {
		name     string
		expr     dst.Expr
		expected string
	}{
		{
			name:     "simple ident",
			expr:     &dst.Ident{Name: "Foo"},
			expected: "Foo",
		},
		{
			name:     "pointer",
			expr:     &dst.StarExpr{X: &dst.Ident{Name: "Foo"}},
			expected: "Foo",
		},
		{
			name: "selector",
			expr: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "pkg"},
				Sel: &dst.Ident{Name: "Type"},
			},
			expected: "Type",
		},
		{
			name: "generic index",
			expr: &dst.IndexExpr{
				X:     &dst.Ident{Name: "Container"},
				Index: &dst.Ident{Name: "T"},
			},
			expected: "Container",
		},
		{
			name: "generic index list",
			expr: &dst.IndexListExpr{
				X:       &dst.Ident{Name: "Map"},
				Indices: []dst.Expr{&dst.Ident{Name: "K"}, &dst.Ident{Name: "V"}},
			},
			expected: "Map",
		},
		{
			name:     "nil",
			expr:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractTypeName(tt.expr); got != tt.expected {
				t.Errorf("ExtractTypeName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractReceiverTypeName(t *testing.T) {
	tests := []struct {
		name     string
		recv     *dst.FieldList
		expected string
	}{
		{
			name:     "nil receiver",
			recv:     nil,
			expected: "",
		},
		{
			name:     "empty list",
			recv:     &dst.FieldList{List: []*dst.Field{}},
			expected: "",
		},
		{
			name: "value receiver",
			recv: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.Ident{Name: "Foo"}},
				},
			},
			expected: "Foo",
		},
		{
			name: "pointer receiver",
			recv: &dst.FieldList{
				List: []*dst.Field{
					{Type: &dst.StarExpr{X: &dst.Ident{Name: "Bar"}}},
				},
			},
			expected: "Bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractReceiverTypeName(tt.recv); got != tt.expected {
				t.Errorf("ExtractReceiverTypeName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestContainsIota(t *testing.T) {
	tests := []struct {
		name     string
		expr     dst.Expr
		expected bool
	}{
		{
			name:     "plain iota",
			expr:     &dst.Ident{Name: "iota"},
			expected: true,
		},
		{
			name:     "not iota",
			expr:     &dst.Ident{Name: "foo"},
			expected: false,
		},
		{
			name: "iota in binary expr",
			expr: &dst.BinaryExpr{
				X:  &dst.Ident{Name: "iota"},
				Op: token.ADD,
				Y:  &dst.BasicLit{Kind: token.INT, Value: "1"},
			},
			expected: true,
		},
		{
			name: "iota in shift",
			expr: &dst.BinaryExpr{
				X:  &dst.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.SHL,
				Y:  &dst.Ident{Name: "iota"},
			},
			expected: true,
		},
		{
			name:     "nil",
			expr:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsIota(tt.expr); got != tt.expected {
				t.Errorf("ContainsIota() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsIotaBlock(t *testing.T) {
	tests := []struct {
		name     string
		decl     *dst.GenDecl
		expected bool
	}{
		{
			name: "const with iota",
			decl: &dst.GenDecl{
				Tok: token.CONST,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names:  []*dst.Ident{{Name: "A"}},
						Values: []dst.Expr{&dst.Ident{Name: "iota"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "const without iota",
			decl: &dst.GenDecl{
				Tok: token.CONST,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names:  []*dst.Ident{{Name: "X"}},
						Values: []dst.Expr{&dst.BasicLit{Kind: token.INT, Value: "1"}},
					},
				},
			},
			expected: false,
		},
		{
			name: "var decl",
			decl: &dst.GenDecl{
				Tok: token.VAR,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names:  []*dst.Ident{{Name: "X"}},
						Values: []dst.Expr{&dst.Ident{Name: "iota"}},
					},
				},
			},
			expected: false,
		},
		{
			name: "type decl",
			decl: &dst.GenDecl{
				Tok:   token.TYPE,
				Specs: []dst.Spec{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIotaBlock(tt.decl); got != tt.expected {
				t.Errorf("IsIotaBlock() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractEnumType(t *testing.T) {
	tests := []struct {
		name     string
		decl     *dst.GenDecl
		expected string
	}{
		{
			name: "typed const",
			decl: &dst.GenDecl{
				Tok: token.CONST,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names:  []*dst.Ident{{Name: "A"}},
						Type:   &dst.Ident{Name: "MyEnum"},
						Values: []dst.Expr{&dst.Ident{Name: "iota"}},
					},
				},
			},
			expected: "MyEnum",
		},
		{
			name: "untyped const",
			decl: &dst.GenDecl{
				Tok: token.CONST,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names:  []*dst.Ident{{Name: "A"}},
						Values: []dst.Expr{&dst.Ident{Name: "iota"}},
					},
				},
			},
			expected: "",
		},
		{
			name: "empty specs",
			decl: &dst.GenDecl{
				Tok:   token.CONST,
				Specs: []dst.Spec{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractEnumType(tt.decl); got != tt.expected {
				t.Errorf("ExtractEnumType() = %q, want %q", got, tt.expected)
			}
		})
	}
}
