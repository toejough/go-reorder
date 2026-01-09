package emit

import (
	"testing"

	"github.com/dave/dst"

	"github.com/toejough/go-reorder/internal/categorize"
)

func TestGetEmitter(t *testing.T) {
	tests := []struct {
		section  string
		expected bool
	}{
		{"imports", true},
		{"main", true},
		{"init", true},
		{"exported_consts", true},
		{"exported_enums", true},
		{"exported_vars", true},
		{"exported_types", true},
		{"exported_funcs", true},
		{"unexported_consts", true},
		{"unexported_enums", true},
		{"unexported_vars", true},
		{"unexported_types", true},
		{"unexported_funcs", true},
		{"uncategorized", true},
		{"bogus_section", false},
	}

	for _, tt := range tests {
		t.Run(tt.section, func(t *testing.T) {
			emitter := GetEmitter(tt.section)
			if tt.expected && emitter == nil {
				t.Errorf("expected emitter for %q, got nil", tt.section)
			}
			if !tt.expected && emitter != nil {
				t.Errorf("expected nil for %q, got emitter", tt.section)
			}
		})
	}
}

func TestEmittersHandleEmpty(t *testing.T) {
	cat := &categorize.CategorizedDecls{}
	cfg := &Config{
		TypeLayout: []string{"typedef", "constructors", "exported_methods", "unexported_methods"},
		EnumLayout: []string{"typedef", "iota", "exported_methods", "unexported_methods"},
	}

	sections := []string{
		"imports", "main", "init",
		"exported_consts", "exported_enums", "exported_vars",
		"exported_types", "exported_funcs",
		"unexported_consts", "unexported_enums", "unexported_vars",
		"unexported_types", "unexported_funcs",
		"uncategorized",
	}

	for _, section := range sections {
		t.Run(section, func(t *testing.T) {
			emitter := GetEmitter(section)
			if emitter == nil {
				t.Skip("no emitter")
			}
			decls := emitter(cat, cfg)
			if decls == nil {
				t.Error("emitter returned nil, expected empty slice")
			}
		})
	}
}

func TestEmitTypeGroup(t *testing.T) {
	tg := &categorize.TypeGroup{
		TypeName: "Server",
		TypeDecl: &dst.GenDecl{},
		Constructors: []*dst.FuncDecl{
			{Name: &dst.Ident{Name: "NewServer"}},
		},
		ExportedMethods: []*dst.FuncDecl{
			{Name: &dst.Ident{Name: "Start"}},
		},
		UnexportedMethods: []*dst.FuncDecl{
			{Name: &dst.Ident{Name: "handleRequest"}},
		},
	}

	tests := []struct {
		name     string
		layout   []string
		expected int
	}{
		{
			name:     "full layout",
			layout:   []string{"typedef", "constructors", "exported_methods", "unexported_methods"},
			expected: 4,
		},
		{
			name:     "typedef only",
			layout:   []string{"typedef"},
			expected: 1,
		},
		{
			name:     "methods only",
			layout:   []string{"exported_methods", "unexported_methods"},
			expected: 2,
		},
		{
			name:     "empty layout",
			layout:   []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decls := EmitTypeGroup(tg, tt.layout)
			if len(decls) != tt.expected {
				t.Errorf("got %d decls, want %d", len(decls), tt.expected)
			}
		})
	}
}

func TestEmitEnumGroup(t *testing.T) {
	eg := &categorize.EnumGroup{
		TypeName:  "Status",
		TypeDecl:  &dst.GenDecl{},
		ConstDecl: &dst.GenDecl{},
		ExportedMethods: []*dst.FuncDecl{
			{Name: &dst.Ident{Name: "String"}},
		},
		UnexportedMethods: []*dst.FuncDecl{
			{Name: &dst.Ident{Name: "isValid"}},
		},
	}

	tests := []struct {
		name     string
		layout   []string
		expected int
	}{
		{
			name:     "full layout",
			layout:   []string{"typedef", "iota", "exported_methods", "unexported_methods"},
			expected: 4,
		},
		{
			name:     "typedef and iota only",
			layout:   []string{"typedef", "iota"},
			expected: 2,
		},
		{
			name:     "empty layout",
			layout:   []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decls := EmitEnumGroup(eg, tt.layout)
			if len(decls) != tt.expected {
				t.Errorf("got %d decls, want %d", len(decls), tt.expected)
			}
		})
	}
}

func TestEmitFuncs(t *testing.T) {
	funcs := []*dst.FuncDecl{
		{Name: &dst.Ident{Name: "Foo"}},
		{Name: &dst.Ident{Name: "Bar"}},
	}

	decls := EmitFuncs(funcs)

	if len(decls) != 2 {
		t.Errorf("got %d decls, want 2", len(decls))
	}

	// Each should have EmptyLine decoration
	for i, decl := range decls {
		fn := decl.(*dst.FuncDecl)
		if fn.Decs.Before != dst.EmptyLine {
			t.Errorf("decl[%d] should have EmptyLine before", i)
		}
	}
}

func TestEmitImports(t *testing.T) {
	t.Run("nil imports", func(t *testing.T) {
		cat := &categorize.CategorizedDecls{Imports: nil}
		decls := Imports(cat)
		if len(decls) != 0 {
			t.Errorf("expected empty, got %d", len(decls))
		}
	})

	t.Run("with imports", func(t *testing.T) {
		cat := &categorize.CategorizedDecls{
			Imports: []dst.Decl{&dst.GenDecl{}},
		}
		decls := Imports(cat)
		if len(decls) != 1 {
			t.Errorf("expected 1, got %d", len(decls))
		}
	})
}

func TestEmitMain(t *testing.T) {
	t.Run("nil main", func(t *testing.T) {
		cat := &categorize.CategorizedDecls{Main: nil}
		decls := Main(cat)
		if len(decls) != 0 {
			t.Errorf("expected empty, got %d", len(decls))
		}
	})

	t.Run("with main", func(t *testing.T) {
		cat := &categorize.CategorizedDecls{
			Main: &dst.FuncDecl{Name: &dst.Ident{Name: "main"}},
		}
		decls := Main(cat)
		if len(decls) != 1 {
			t.Errorf("expected 1, got %d", len(decls))
		}
	})
}
