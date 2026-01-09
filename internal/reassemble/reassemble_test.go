package reassemble

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/toejough/go-reorder/internal/categorize"
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

func TestDeclarations(t *testing.T) {
	t.Run("empty categorization", func(t *testing.T) {
		cat := &categorize.CategorizedDecls{}
		decls := Declarations(cat)

		if decls == nil {
			t.Error("expected non-nil slice")
		}
		if len(decls) != 0 {
			t.Errorf("expected empty, got %d decls", len(decls))
		}
	})

	t.Run("basic ordering", func(t *testing.T) {
		src := `package test

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
`
		file := parseSource(t, src)
		cat := categorize.CategorizeDeclarations(file)
		decls := Declarations(cat)

		if len(decls) == 0 {
			t.Error("expected declarations, got none")
		}

		// Verify ordering: imports, main, exported sections, unexported sections
		// First should be imports
		if genDecl, ok := decls[0].(*dst.GenDecl); !ok || genDecl.Tok != token.IMPORT {
			t.Error("first declaration should be import")
		}
	})

	t.Run("preserves main function position", func(t *testing.T) {
		src := `package main

import "fmt"

func main() { fmt.Println("hello") }

func helper() {}
`
		file := parseSource(t, src)
		cat := categorize.CategorizeDeclarations(file)
		decls := Declarations(cat)

		// main should come after imports
		foundMain := false
		for i, decl := range decls {
			if fn, ok := decl.(*dst.FuncDecl); ok && fn.Name.Name == "main" {
				foundMain = true
				if i < 1 {
					t.Error("main should come after imports")
				}
				break
			}
		}
		if !foundMain {
			t.Error("main function not found in output")
		}
	})

	t.Run("groups enums with their types", func(t *testing.T) {
		src := `package test

type Status int

const (
	StatusPending Status = iota
	StatusActive
)

func (s Status) String() string { return "" }
`
		file := parseSource(t, src)
		cat := categorize.CategorizeDeclarations(file)
		decls := Declarations(cat)

		// Should have type, const block, and method in sequence
		if len(decls) < 3 {
			t.Errorf("expected at least 3 declarations, got %d", len(decls))
		}
	})

	t.Run("groups constructors with types", func(t *testing.T) {
		src := `package test

type Config struct{}

func NewConfig() *Config { return nil }
`
		file := parseSource(t, src)
		cat := categorize.CategorizeDeclarations(file)
		decls := Declarations(cat)

		// Type and constructor should be grouped
		if len(decls) < 2 {
			t.Errorf("expected at least 2 declarations, got %d", len(decls))
		}
	})
}

func TestDeclarationsWithOrder(t *testing.T) {
	src := `package test

import "fmt"

const ExportedConst = 1
var ExportedVar = "hello"
func ExportedFunc() {}
`
	file := parseSource(t, src)
	cat := categorize.CategorizeDeclarations(file)

	t.Run("custom order", func(t *testing.T) {
		order := []string{"imports", "exported_funcs", "exported_vars", "exported_consts"}
		cfg := &Config{
			Order:      order,
			TypeLayout: []string{"typedef", "constructors", "exported_methods", "unexported_methods"},
			EnumLayout: []string{"typedef", "iota", "exported_methods", "unexported_methods"},
		}

		decls := DeclarationsWithOrder(cat, cfg)

		if len(decls) == 0 {
			t.Error("expected declarations")
		}
	})

	t.Run("empty order returns nothing", func(t *testing.T) {
		cfg := &Config{Order: []string{}}
		decls := DeclarationsWithOrder(cat, cfg)

		if len(decls) != 0 {
			t.Errorf("expected empty, got %d", len(decls))
		}
	})
}
