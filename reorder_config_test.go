package reorder_test

import (
	"testing"

	"github.com/toejough/go-reorder"
)

func TestSourceWithConfig_DefaultMatchesOriginal(t *testing.T) {
	input := `package example

func helper() {}

const Version = "1.0"

type Config struct{}

func main() {}
`
	// Get output from original API
	original, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source failed: %v", err)
	}

	// Get output from config API with defaults
	cfg := reorder.DefaultConfig()
	withConfig, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	if original != withConfig {
		t.Errorf("SourceWithConfig with defaults differs from Source:\n--- Original ---\n%s\n--- WithConfig ---\n%s", original, withConfig)
	}
}

func TestSourceWithConfig_CustomOrder(t *testing.T) {
	input := `package example

import "fmt"

func main() { fmt.Println("hi") }

const Version = "1.0"

func Helper() {}
`
	// Custom order: funcs before consts
	cfg := reorder.DefaultConfig()
	cfg.Sections.Order = []string{
		"imports",
		"main",
		"exported_funcs",
		"exported_consts",
		"uncategorized",
	}

	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Verify Helper appears before Version in output
	if !containsInOrder(result, "func Helper()", "Version") {
		t.Errorf("Expected Helper before Version, got:\n%s", result)
	}
}

func TestSourceWithConfig_Idempotent(t *testing.T) {
	input := `package example

func helper() {}

const Version = "1.0"

type Config struct{}
`
	cfg := reorder.DefaultConfig()

	first, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("first SourceWithConfig failed: %v", err)
	}

	second, err := reorder.SourceWithConfig(first, cfg)
	if err != nil {
		t.Fatalf("second SourceWithConfig failed: %v", err)
	}

	if first != second {
		t.Errorf("SourceWithConfig not idempotent:\n--- First ---\n%s\n--- Second ---\n%s", first, second)
	}
}

// containsInOrder checks if a appears before b in s.
func containsInOrder(s, a, b string) bool {
	idxA := indexOf(s, a)
	idxB := indexOf(s, b)
	return idxA != -1 && idxB != -1 && idxA < idxB
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSourceWithConfig_TypeLayout_MethodsBeforeConstructors(t *testing.T) {
	input := `package example

type Server struct{}

func NewServer() *Server { return &Server{} }

func (s *Server) Start() {}
`
	cfg := reorder.DefaultConfig()
	// Put methods before constructors
	cfg.Types.TypeLayout = []string{
		"typedef",
		"exported_methods",
		"unexported_methods",
		"constructors",
	}

	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Verify Start() appears before NewServer in output
	if !containsInOrder(result, "func (s *Server) Start()", "func NewServer()") {
		t.Errorf("Expected Start before NewServer, got:\n%s", result)
	}
}

func TestSourceWithConfig_EnumLayout_IotaBeforeTypedef(t *testing.T) {
	input := `package example

type Status int

const (
	StatusPending Status = iota
	StatusActive
)
`
	cfg := reorder.DefaultConfig()
	// Put iota before typedef
	cfg.Types.EnumLayout = []string{
		"iota",
		"typedef",
		"exported_methods",
		"unexported_methods",
	}

	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Verify const block appears before type declaration in output
	if !containsInOrder(result, "StatusPending", "type Status int") {
		t.Errorf("Expected iota const before typedef, got:\n%s", result)
	}
}

func TestSourceWithConfig_InitFunctions(t *testing.T) {
	input := `package example

func helper() {}

func init() {
	// first init
}

const Version = "1.0"

func init() {
	// second init
}
`
	cfg := reorder.DefaultConfig()
	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Verify both init functions are present
	if !hasSubstring(result, "// first init") {
		t.Error("first init function missing")
	}
	if !hasSubstring(result, "// second init") {
		t.Error("second init function missing")
	}

	// Verify init functions maintain relative order
	if !containsInOrder(result, "// first init", "// second init") {
		t.Errorf("init functions not in original order:\n%s", result)
	}
}

func TestSourceWithConfig_InitAfterMain(t *testing.T) {
	input := `package example

func init() {}

func main() {}
`
	cfg := reorder.DefaultConfig()
	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Default order: main comes before init
	if !containsInOrder(result, "func main()", "func init()") {
		t.Errorf("Expected main before init:\n%s", result)
	}
}

func TestSourceWithConfig_ModeAppend(t *testing.T) {
	input := `package example

func Helper() {}

const Version = "1.0"
`
	// Config that only includes imports - Helper and Version are "uncategorized"
	cfg := reorder.DefaultConfig()
	cfg.Sections.Order = []string{"imports", "uncategorized"}
	cfg.Behavior.Mode = "append"

	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Both should be present in uncategorized
	if !hasSubstring(result, "func Helper()") {
		t.Error("Helper function missing")
	}
	if !hasSubstring(result, "Version") {
		t.Error("Version const missing")
	}
}

func TestSourceWithConfig_ModeDrop(t *testing.T) {
	input := `package example

func Helper() {}

const Version = "1.0"
`
	// Config that only includes exported_funcs - Version const should be dropped
	cfg := reorder.DefaultConfig()
	cfg.Sections.Order = []string{"exported_funcs"}
	cfg.Behavior.Mode = "drop"

	result, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("SourceWithConfig failed: %v", err)
	}

	// Helper should be present
	if !hasSubstring(result, "func Helper()") {
		t.Error("Helper function missing")
	}
	// Version should be dropped
	if hasSubstring(result, "Version") {
		t.Error("Version const should have been dropped")
	}
}

func hasSubstring(s, sub string) bool {
	return indexOf(s, sub) != -1
}