package categorize

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func parseSource(t *testing.T, src string) *dst.File {
	t.Helper()

	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	return file
}

func TestCategorizeDeclarations(t *testing.T) {
	tests := []struct {
		name                 string
		src                  string
		expectedMain         bool
		expectedInitCount    int
		expectedExpConsts    int
		expectedUnexpConsts  int
		expectedExpVars      int
		expectedUnexpVars    int
		expectedExpTypes     int
		expectedUnexpTypes   int
		expectedExpEnums     int
		expectedUnexpEnums   int
		expectedExpFuncs     int
		expectedUnexpFuncs   int
	}{
		{
			name: "basic categorization",
			src: `package test

import "fmt"

const ExportedConst = 1
const unexportedConst = 2

var ExportedVar = "hello"
var unexportedVar = "world"

type ExportedType struct{}
type unexportedType struct{}

func ExportedFunc() {}
func unexportedFunc() {}

func main() {}
`,
			expectedMain:        true,
			expectedExpConsts:   1,
			expectedUnexpConsts: 1,
			expectedExpVars:     1,
			expectedUnexpVars:   1,
			expectedExpTypes:    1,
			expectedUnexpTypes:  1,
			expectedExpFuncs:    1,
			expectedUnexpFuncs:  1,
		},
		{
			name: "enum detection",
			src: `package test

type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusDone
)
`,
			expectedExpTypes: 0, // Type is grouped with enum
			expectedExpEnums: 1,
		},
		{
			name: "unexported enum",
			src: `package test

type status int

const (
	statusPending status = iota
	statusActive
)
`,
			expectedUnexpTypes: 0,
			expectedUnexpEnums: 1,
		},
		{
			name: "init functions",
			src: `package test

func init() {}
func init() {}
`,
			expectedInitCount: 2,
		},
		{
			name: "method grouping with type",
			src: `package test

type Server struct{}

func (s *Server) Start() {}
func (s *Server) stop() {}
`,
			expectedExpTypes: 1,
		},
		{
			name: "constructor grouping",
			src: `package test

type Config struct{}

func NewConfig() *Config { return nil }
func NewConfigWithTimeout() *Config { return nil }
`,
			expectedExpTypes: 1,
			expectedExpFuncs: 0, // Constructors grouped with type
		},
		{
			name: "untyped iota block treated as constants not enum",
			src: `package test

const (
	a = iota
	b
	c
)
`,
			expectedUnexpConsts: 3, // Should be regular constants, not an enum
			expectedUnexpEnums:  0, // No enum since no type annotation
		},
		{
			name: "exported untyped iota block treated as constants",
			src: `package test

const (
	A = iota
	B
	C
)
`,
			expectedExpConsts:  3,
			expectedUnexpEnums: 0,
			expectedExpEnums:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := parseSource(t, tt.src)
			cat := CategorizeDeclarations(file)

			if tt.expectedMain && cat.Main == nil {
				t.Error("expected main function, got nil")
			}
			if !tt.expectedMain && cat.Main != nil {
				t.Error("expected no main function, got one")
			}

			if len(cat.Init) != tt.expectedInitCount {
				t.Errorf("init count = %d, want %d", len(cat.Init), tt.expectedInitCount)
			}

			if len(cat.ExportedConsts) != tt.expectedExpConsts {
				t.Errorf("exported consts = %d, want %d", len(cat.ExportedConsts), tt.expectedExpConsts)
			}
			if len(cat.UnexportedConsts) != tt.expectedUnexpConsts {
				t.Errorf("unexported consts = %d, want %d", len(cat.UnexportedConsts), tt.expectedUnexpConsts)
			}

			if len(cat.ExportedVars) != tt.expectedExpVars {
				t.Errorf("exported vars = %d, want %d", len(cat.ExportedVars), tt.expectedExpVars)
			}
			if len(cat.UnexportedVars) != tt.expectedUnexpVars {
				t.Errorf("unexported vars = %d, want %d", len(cat.UnexportedVars), tt.expectedUnexpVars)
			}

			if len(cat.ExportedTypes) != tt.expectedExpTypes {
				t.Errorf("exported types = %d, want %d", len(cat.ExportedTypes), tt.expectedExpTypes)
			}
			if len(cat.UnexportedTypes) != tt.expectedUnexpTypes {
				t.Errorf("unexported types = %d, want %d", len(cat.UnexportedTypes), tt.expectedUnexpTypes)
			}

			if len(cat.ExportedEnums) != tt.expectedExpEnums {
				t.Errorf("exported enums = %d, want %d", len(cat.ExportedEnums), tt.expectedExpEnums)
			}
			if len(cat.UnexportedEnums) != tt.expectedUnexpEnums {
				t.Errorf("unexported enums = %d, want %d", len(cat.UnexportedEnums), tt.expectedUnexpEnums)
			}

			if len(cat.ExportedFuncs) != tt.expectedExpFuncs {
				t.Errorf("exported funcs = %d, want %d", len(cat.ExportedFuncs), tt.expectedExpFuncs)
			}
			if len(cat.UnexportedFuncs) != tt.expectedUnexpFuncs {
				t.Errorf("unexported funcs = %d, want %d", len(cat.UnexportedFuncs), tt.expectedUnexpFuncs)
			}
		})
	}
}

func TestIdentifySection(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name:     "import",
			src:      `package test; import "fmt"`,
			expected: "Imports",
		},
		{
			name:     "main function",
			src:      `package test; func main() {}`,
			expected: "main()",
		},
		{
			name:     "exported const",
			src:      `package test; const Foo = 1`,
			expected: "Exported Constants",
		},
		{
			name:     "unexported const",
			src:      `package test; const foo = 1`,
			expected: "unexported constants",
		},
		{
			name:     "exported var",
			src:      `package test; var Foo = 1`,
			expected: "Exported Variables",
		},
		{
			name:     "unexported var",
			src:      `package test; var foo = 1`,
			expected: "unexported variables",
		},
		{
			name:     "exported type",
			src:      `package test; type Foo struct{}`,
			expected: "Exported Types",
		},
		{
			name:     "unexported type",
			src:      `package test; type foo struct{}`,
			expected: "unexported types",
		},
		{
			name:     "exported func",
			src:      `package test; func Foo() {}`,
			expected: "Exported Functions",
		},
		{
			name:     "unexported func",
			src:      `package test; func foo() {}`,
			expected: "unexported functions",
		},
		{
			name:     "exported enum",
			src:      `package test; type Status int; const (StatusA Status = iota)`,
			expected: "Exported Enums",
		},
		{
			name:     "unexported enum",
			src:      `package test; type status int; const (statusA status = iota)`,
			expected: "unexported enums",
		},
		{
			name:     "exported method",
			src:      `package test; type Foo struct{}; func (f *Foo) Bar() {}`,
			expected: "Exported Types",
		},
		{
			name:     "unexported type method",
			src:      `package test; type foo struct{}; func (f *foo) bar() {}`,
			expected: "unexported types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := parseSource(t, tt.src)

			// Find the non-import declaration to test
			var decl dst.Decl
			for _, d := range file.Decls {
				if genDecl, ok := d.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
					if tt.expected == "Imports" {
						decl = d
						break
					}
					continue
				}
				decl = d
			}

			if decl == nil {
				t.Fatal("no declaration found")
			}

			got := IdentifySection(decl)
			if got != tt.expected {
				t.Errorf("IdentifySection() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSortCategorized(t *testing.T) {
	src := `package test

const C = 1
const A = 2
const B = 3

var Z = 1
var X = 2
var Y = 3

type Zebra struct{}
type Apple struct{}
type Banana struct{}

func Zulu() {}
func Alpha() {}
func Beta() {}
`

	file := parseSource(t, src)
	cat := CategorizeDeclarations(file)

	// Verify sorted order
	if len(cat.ExportedConsts) != 3 {
		t.Fatalf("expected 3 consts, got %d", len(cat.ExportedConsts))
	}

	expectedConsts := []string{"A", "B", "C"}
	for i, name := range expectedConsts {
		if cat.ExportedConsts[i].Names[0].Name != name {
			t.Errorf("const[%d] = %s, want %s", i, cat.ExportedConsts[i].Names[0].Name, name)
		}
	}

	expectedVars := []string{"X", "Y", "Z"}
	for i, name := range expectedVars {
		if cat.ExportedVars[i].Names[0].Name != name {
			t.Errorf("var[%d] = %s, want %s", i, cat.ExportedVars[i].Names[0].Name, name)
		}
	}

	expectedTypes := []string{"Apple", "Banana", "Zebra"}
	for i, name := range expectedTypes {
		if cat.ExportedTypes[i].TypeName != name {
			t.Errorf("type[%d] = %s, want %s", i, cat.ExportedTypes[i].TypeName, name)
		}
	}

	expectedFuncs := []string{"Alpha", "Beta", "Zulu"}
	for i, name := range expectedFuncs {
		if cat.ExportedFuncs[i].Name.Name != name {
			t.Errorf("func[%d] = %s, want %s", i, cat.ExportedFuncs[i].Name.Name, name)
		}
	}
}

func TestCollectUncategorized(t *testing.T) {
	src := `package test

const ExportedConst = 1
var ExportedVar = 2
type ExportedType struct{}
func ExportedFunc() {}
`

	file := parseSource(t, src)
	cat := CategorizeDeclarations(file)

	// Exclude exported_consts section
	includedSections := map[string]bool{
		"exported_consts": false,
		"exported_vars":   true,
		"exported_types":  true,
		"exported_funcs":  true,
	}

	CollectUncategorized(cat, includedSections)

	if len(cat.ExportedConsts) != 0 {
		t.Errorf("exported consts should be empty after collect, got %d", len(cat.ExportedConsts))
	}

	if len(cat.Uncategorized) != 1 {
		t.Errorf("uncategorized should have 1 item, got %d", len(cat.Uncategorized))
	}
}

func TestTypeGroup(t *testing.T) {
	src := `package test

type Server struct{}

func NewServer() *Server { return nil }
func (s *Server) Start() {}
func (s *Server) Stop() {}
func (s *Server) handleRequest() {}
`

	file := parseSource(t, src)
	cat := CategorizeDeclarations(file)

	if len(cat.ExportedTypes) != 1 {
		t.Fatalf("expected 1 type group, got %d", len(cat.ExportedTypes))
	}

	tg := cat.ExportedTypes[0]

	if tg.TypeName != "Server" {
		t.Errorf("type name = %s, want Server", tg.TypeName)
	}

	if len(tg.Constructors) != 1 {
		t.Errorf("constructors = %d, want 1", len(tg.Constructors))
	}

	if len(tg.ExportedMethods) != 2 {
		t.Errorf("exported methods = %d, want 2", len(tg.ExportedMethods))
	}

	if len(tg.UnexportedMethods) != 1 {
		t.Errorf("unexported methods = %d, want 1", len(tg.UnexportedMethods))
	}
}

func TestEnumGroup(t *testing.T) {
	src := `package test

type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusDone
)

func (s Status) String() string { return "" }
func (s Status) isValid() bool { return true }
`

	file := parseSource(t, src)
	cat := CategorizeDeclarations(file)

	if len(cat.ExportedEnums) != 1 {
		t.Fatalf("expected 1 enum group, got %d", len(cat.ExportedEnums))
	}

	eg := cat.ExportedEnums[0]

	if eg.TypeName != "Status" {
		t.Errorf("type name = %s, want Status", eg.TypeName)
	}

	if eg.TypeDecl == nil {
		t.Error("type decl should not be nil")
	}

	if eg.ConstDecl == nil {
		t.Error("const decl should not be nil")
	}

	if len(eg.ExportedMethods) != 1 {
		t.Errorf("exported methods = %d, want 1", len(eg.ExportedMethods))
	}

	if len(eg.UnexportedMethods) != 1 {
		t.Errorf("unexported methods = %d, want 1", len(eg.UnexportedMethods))
	}
}
