package reorder_test

import (
	"testing"

	"github.com/toejough/go-reorder"
)

//nolint:cyclop,funlen,gocognit // Test function with validation logic; complexity from comprehensive test cases
func TestAnalyzeSectionOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *reorder.SectionOrder)
	}{
		{
			name: "correctly ordered code",
			input: `package example

import "fmt"

func main() {}

const Version = "1.0"

type Config struct{}

func Start() {}
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()

				if len(order.Sections) == 0 {
					t.Error("Expected non-empty sections")
				}
				// Verify sections are in ascending order by position
				for i := 1; i < len(order.Sections); i++ {
					if order.Sections[i].Position < order.Sections[i-1].Position {
						t.Errorf("Sections not in position order: %v", order.Sections)
					}
				}
			},
		},
		{
			name: "incorrectly ordered code",
			input: `package example

func helper() {}

const Version = "1.0"

type Config struct{}

func main() {}
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()
				// Find main() section
				var mainSection *reorder.Section

				for i := range order.Sections {
					if order.Sections[i].Name == "main()" {
						mainSection = &order.Sections[i]
						break
					}
				}

				if mainSection == nil {
					t.Error("Expected to find main() section")
					return
				}
				// main() should be at position 4 but expected at position 2
				if mainSection.Position == mainSection.Expected {
					t.Error("Expected main() to be out of order")
				}
			},
		},
		{
			name: "code with imports and multiple sections",
			input: `package example

import "fmt"

const maxRetries = 10

type server struct{}

func (s *server) start() {}

const Version = "1.0"

func NewServer() *server { return &server{} }
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()

				sectionNames := make(map[string]bool)
				for _, sec := range order.Sections {
					sectionNames[sec.Name] = true
				}
				// Should have both exported and unexported sections
				expectedSections := []string{"Imports", "Exported Constants", "unexported constants", "unexported types"}
				for _, expected := range expectedSections {
					if !sectionNames[expected] {
						t.Errorf("Expected section %q not found", expected)
					}
				}
			},
		},
		{
			name: "invalid Go code",
			input: `package example

func 123InvalidName() {}`,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			order, err := reorder.AnalyzeSectionOrder(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AnalyzeSectionOrder() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}

			if err == nil && testCase.validate != nil {
				testCase.validate(t, order)
			}
		})
	}
}

func TestSource_BasicReordering(t *testing.T) {
	t.Parallel()

	input := `package example

func helper() {}

const Version = "1.0"

type Config struct {}

func main() {}

var Debug = false
`

	expected := `package example

func main() {}

// Exported constants.
const (
	Version = "1.0"
)

// Exported variables.
var (
	Debug = false
)

type Config struct{}

func helper() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_ConstructorGrouping(t *testing.T) {
	t.Parallel()

	input := `package example

func NewConfig() *Config {
	return &Config{}
}

type Config struct {
	timeout int
}

func (c *Config) Validate() error {
	return nil
}

func NewConfigWithTimeout(t int) *Config {
	return &Config{timeout: t}
}
`

	expected := `package example

type Config struct {
	timeout int
}

func NewConfig() *Config {
	return &Config{}
}

func NewConfigWithTimeout(t int) *Config {
	return &Config{timeout: t}
}

func (c *Config) Validate() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_ConstructorWithPrefix(t *testing.T) {
	t.Parallel()

	input := `package example

type FileOps struct {
	value int
}

func NewFileOps(v int) *FileOps {
	return &FileOps{value: v}
}

func (f *FileOps) DoSomething() error {
	return nil
}

func NewRealFileOps() *FileOps {
	return &FileOps{value: 42}
}

func NewMockFileOps() *FileOps {
	return &FileOps{value: 0}
}
`

	expected := `package example

type FileOps struct {
	value int
}

func NewFileOps(v int) *FileOps {
	return &FileOps{value: v}
}

func NewMockFileOps() *FileOps {
	return &FileOps{value: 0}
}

func NewRealFileOps() *FileOps {
	return &FileOps{value: 42}
}

func (f *FileOps) DoSomething() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_EnumHandling(t *testing.T) {
	t.Parallel()

	input := `package example

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

type Status int

const MaxRetries = 3
`

	expected := `package example

// Exported constants.
const (
	MaxRetries = 3
)

type Status int

// Status values.
const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_ExportedUnexportedSeparation(t *testing.T) {
	t.Parallel()

	input := `package example

const maxWorkers = 10

var Debug = false

const Version = "1.0"

func helper() {}

func Start() {}

var workers = 0
`

	expected := `package example

// Exported constants.
const (
	Version = "1.0"
)

// Exported variables.
var (
	Debug = false
)

func Start() {}

// unexported constants.
const (
	maxWorkers = 10
)

// unexported variables.
var (
	workers = 0
)

func helper() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_Idempotency(t *testing.T) {
	t.Parallel()

	input := `package example

type Config struct{}

func NewConfig() *Config { return &Config{} }

const Version = "1.0"
`

	first, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("First Source() error = %v", err)
	}

	second, err := reorder.Source(first)
	if err != nil {
		t.Fatalf("Second Source() error = %v", err)
	}

	if first != second {
		t.Errorf("Not idempotent:\nFirst:\n%s\n\nSecond:\n%s", first, second)
	}
}

func TestSource_MethodOrdering(t *testing.T) {
	t.Parallel()

	input := `package example

type Server struct{}

func (s *Server) start() {}

func (s *Server) Stop() {}

func (s *Server) Start() {}

func (s *Server) shutdown() {}
`

	expected := `package example

type Server struct{}

func (s *Server) Start() {}

func (s *Server) Stop() {}

func (s *Server) shutdown() {}

func (s *Server) start() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_MultipleEnums(t *testing.T) {
	t.Parallel()

	input := `package example

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

type Status int

const (
	StatusPending Status = iota
	StatusActive
)

type Priority int
`

	expected := `package example

type Priority int

// Priority values.
const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

type Status int

// Status values.
const (
	StatusPending Status = iota
	StatusActive
)
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_EnumWithMethods is a regression test for the bug where methods
// on enum types were being deleted during reordering.
// This bug occurred because:
// 1. Enum types are identified and moved to enumGroup structs
// 2. Methods were associated with types in typeGroups map
// 3. When enums were extracted from regular types, methods stayed in typeGroups
// 4. enumGroup didn't have a methods field, so methods were never output
func TestSource_EnumWithMethods(t *testing.T) {
	t.Parallel()

	input := `package example

type ChangeType int

const (
	MonotonicCount ChangeType = iota
	FluctuatingCount
	Content
)

func (ct *ChangeType) String() string {
	switch *ct {
	case MonotonicCount:
		return "monotonic"
	case FluctuatingCount:
		return "fluctuating"
	case Content:
		return "content"
	default:
		return "unknown"
	}
}

func (ct *ChangeType) UnmarshalText(text []byte) error {
	return nil
}
`

	expected := `package example

type ChangeType int

// ChangeType values.
const (
	MonotonicCount ChangeType = iota
	FluctuatingCount
	Content
)

func (ct *ChangeType) String() string {
	switch *ct {
	case MonotonicCount:
		return "monotonic"
	case FluctuatingCount:
		return "fluctuating"
	case Content:
		return "content"
	default:
		return "unknown"
	}
}

func (ct *ChangeType) UnmarshalText(text []byte) error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_EnumWithMixedMethodVisibility tests that enum types preserve
// both exported and unexported methods in correct order.
func TestSource_EnumWithMixedMethodVisibility(t *testing.T) {
	t.Parallel()

	input := `package example

const (
	StatusActive Status = iota
	StatusInactive
)

type Status int

func (s *Status) validate() error {
	return nil
}

func (s *Status) String() string {
	return "status"
}

func (s *Status) IsActive() bool {
	return *s == StatusActive
}
`

	expected := `package example

type Status int

// Status values.
const (
	StatusActive Status = iota
	StatusInactive
)

func (s *Status) IsActive() bool {
	return *s == StatusActive
}

func (s *Status) String() string {
	return "status"
}

func (s *Status) validate() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_MethodsOnlyFile is a regression test for the bug where files containing
// only receiver methods (no type definition, no constants, no vars, no package-level functions)
// would have all methods deleted after reorder.Source() processing.
//
// This bug occurred because:
// 1. categorizeDeclarations creates typeGroups for methods even when type definition is absent
// 2. typeGroup.typeDecl remains nil when no type declaration exists in the file
// 3. reassembleDeclarations has logic: if typeGrp.typeDecl != nil { add typeDecl }
// 4. But the methods are added in the SAME loop iteration after the type declaration
// 5. If typeDecl is nil, we never enter the condition, and continue to next iteration
// 6. Methods are never added to output, resulting in deletion
//
// Expected behavior: Methods should be preserved even when their type is defined elsewhere.
// This is common in Go projects where types are defined in one file and methods are
// spread across multiple files for organization.
func TestSource_MethodsOnlyFile(t *testing.T) {
	t.Parallel()

	// Simulate a file that only contains methods for Engine type (defined elsewhere)
	input := `package example

func (e *Engine) SetSourceResizable(resizable bool) {
	e.sourceResizable = resizable
}

func (e *Engine) Start() error {
	return nil
}

func (e *Engine) stop() {
	// internal cleanup
}
`

	// Expected: Methods should be preserved and reordered (exported first, then unexported, alphabetically)
	expected := `package example

func (e *Engine) SetSourceResizable(resizable bool) {
	e.sourceResizable = resizable
}

func (e *Engine) Start() error {
	return nil
}

func (e *Engine) stop() {
	// internal cleanup
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_MethodsOnlyFile_MultipleTypes tests the case where a file contains
// methods for multiple types (all defined elsewhere).
func TestSource_MethodsOnlyFile_MultipleTypes(t *testing.T) {
	t.Parallel()

	input := `package example

func (s *Server) Stop() error {
	return nil
}

func (c *Client) Connect() error {
	return nil
}

func (s *Server) Start() error {
	return nil
}

func (c *Client) disconnect() {
	// cleanup
}
`

	// Expected: Methods grouped by type, each type's methods sorted (exported first, then unexported)
	// Since we don't have type definitions, types should be sorted alphabetically: Client, Server
	expected := `package example

func (c *Client) Connect() error {
	return nil
}

func (c *Client) disconnect() {
	// cleanup
}

func (s *Server) Start() error {
	return nil
}

func (s *Server) Stop() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_MethodsOnlyFile_ExportedType tests that methods on exported types
// (defined elsewhere) are categorized correctly as exported type methods.
func TestSource_MethodsOnlyFile_ExportedType(t *testing.T) {
	t.Parallel()

	input := `package example

func (e *Engine) shutdown() {
	// internal
}

func (e *Engine) Start() error {
	return nil
}
`

	expected := `package example

func (e *Engine) Start() error {
	return nil
}

func (e *Engine) shutdown() {
	// internal
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_MethodsOnlyFile_UnexportedType tests that methods on unexported types
// (defined elsewhere) are also preserved and categorized correctly.
func TestSource_MethodsOnlyFile_UnexportedType(t *testing.T) {
	t.Parallel()

	input := `package example

func (e *engine) start() error {
	return nil
}

func (e *engine) Stop() error {
	return nil
}
`

	// Methods on unexported type should still be grouped together
	expected := `package example

func (e *engine) Stop() error {
	return nil
}

func (e *engine) start() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_MixedMethodsAndFunctions tests that a file with both methods (type defined elsewhere)
// and standalone functions works correctly.
func TestSource_MixedMethodsAndFunctions(t *testing.T) {
	t.Parallel()

	input := `package example

func (e *Engine) Start() error {
	return nil
}

func Helper() {
	// standalone function
}

func (e *Engine) stop() {
	// internal
}
`

	expected := `package example

func (e *Engine) Start() error {
	return nil
}

func (e *Engine) stop() {
	// internal
}

func Helper() {
	// standalone function
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestSource_GlowsyncRealCase tests the exact file content from glowsync that was reported as failing.
// This is a regression test for the real-world case that motivated the bug fix.
func TestSource_GlowsyncRealCase(t *testing.T) {
	t.Parallel()

	input := `package syncengine

import (
	"sync/atomic"

	"github.com/joe/copy-files/pkg/filesystem"
)

// SetSourceResizable injects a mock ResizablePool for source filesystem (test helper).
// Used by refactored tests that use observable behavior testing via ResizablePool mocks.
func (e *Engine) SetSourceResizable(pool filesystem.ResizablePool) {
	e.sourceResizable = pool
}

// SetDesiredWorkers sets the desired worker count (test helper for initialization).
// Used to initialize test state before triggering scaling decisions.
func (e *Engine) SetDesiredWorkers(count int) {
	atomic.StoreInt32(&e.desiredWorkers, int32(count))
}

// GetDesiredWorkers returns the current desired worker count (test helper).
// Only used in 6 legitimate cases where observable behavior testing isn't applicable:
// - Timing tests that detect when evaluation occurred
// - Boundary tests that verify min/max worker bounds
// - Non-deterministic tests with random perturbation
func (e *Engine) GetDesiredWorkers() int32 {
	return atomic.LoadInt32(&e.desiredWorkers)
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	// The methods should be preserved (in alphabetical order: Get, Set, Set)
	if len(result) < 500 {
		t.Errorf("Result suspiciously short (%d bytes), methods may have been deleted:\n%s", len(result), result)
	}

	// Check that all three methods are present in the result
	methods := []string{
		"func (e *Engine) SetSourceResizable",
		"func (e *Engine) SetDesiredWorkers",
		"func (e *Engine) GetDesiredWorkers",
	}
	for _, method := range methods {
		if !contains(result, method) {
			t.Errorf("Method %q not found in result:\n%s", method, result)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

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

func TestSource_ParseErrorIncludesPosition(t *testing.T) {
	t.Parallel()

	// Invalid Go code with error on line 3
	input := `package example

func 123InvalidName() {}
`

	_, err := reorder.Source(input)
	if err == nil {
		t.Fatal("Expected error for invalid Go code")
	}

	errMsg := err.Error()
	// Error should include line number in format "line X" or "X:Y:" (line:column)
	if !hasSubstring(errMsg, "line 3") && !hasSubstring(errMsg, "3:") {
		t.Errorf("Error message should include line number, got: %s", errMsg)
	}
}

func TestStrictModeError(t *testing.T) {
	t.Parallel()

	input := `package example

const Version = "1.0"

func Helper() {}
`

	// Config that doesn't include exported_consts or exported_funcs
	cfg := &reorder.Config{
		Sections: reorder.SectionsConfig{
			Order: []string{"imports", "main"},
		},
		Types: reorder.TypesConfig{
			TypeLayout: []string{"typedef", "constructors", "exported_methods", "unexported_methods"},
			EnumLayout: []string{"typedef", "iota", "exported_methods", "unexported_methods"},
		},
		Behavior: reorder.BehaviorConfig{
			Mode: "strict",
		},
	}

	_, err := reorder.SourceWithConfig(input, cfg)
	if err == nil {
		t.Fatal("Expected strict mode error")
	}

	errMsg := err.Error()

	// Should mention the excluded sections
	if !hasSubstring(errMsg, "exported_consts") {
		t.Errorf("Error should mention exported_consts, got: %s", errMsg)
	}
	if !hasSubstring(errMsg, "exported_funcs") {
		t.Errorf("Error should mention exported_funcs, got: %s", errMsg)
	}

	// Should include hints
	if !hasSubstring(errMsg, "Hints:") {
		t.Errorf("Error should include hints, got: %s", errMsg)
	}
	if !hasSubstring(errMsg, "uncategorized") {
		t.Errorf("Error should suggest uncategorized, got: %s", errMsg)
	}
	if !hasSubstring(errMsg, "--mode=append") {
		t.Errorf("Error should suggest --mode=append, got: %s", errMsg)
	}
}

func TestStrictModeNoErrorWhenConfigComplete(t *testing.T) {
	t.Parallel()

	input := `package example

const Version = "1.0"

func Helper() {}
`

	// Config that includes all needed sections
	cfg := reorder.DefaultConfig()
	cfg.Behavior.Mode = "strict"

	_, err := reorder.SourceWithConfig(input, cfg)
	if err != nil {
		t.Fatalf("Should not error when config is complete: %v", err)
	}
}

func TestVarBlockTrailingCommentsPreserved(t *testing.T) {
	// Issue #2: Trailing comments on var declarations get misplaced
	// The comment from the first declaration was ending up on the closing paren
	t.Parallel()

	input := `package main

var (
	zebra bool //nolint:something // zebra comment
	apple bool //nolint:other // apple comment
)

func main() {
	_ = zebra
	_ = apple
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source failed: %v", err)
	}

	// After reordering, apple should come before zebra (alphabetical)
	// Each variable should keep its trailing comment

	// Check that zebra's comment stays with zebra
	if !hasSubstring(result, "zebra bool //nolint:something // zebra comment") {
		t.Errorf("zebra's trailing comment was not preserved.\nGot:\n%s", result)
	}

	// Check that apple's comment stays with apple
	if !hasSubstring(result, "apple bool //nolint:other // apple comment") {
		t.Errorf("apple's trailing comment was not preserved.\nGot:\n%s", result)
	}

	// The closing paren should NOT have a trailing comment
	if hasSubstring(result, ") //nolint") {
		t.Errorf("trailing comment incorrectly moved to closing paren.\nGot:\n%s", result)
	}
}

func TestGroupedTypeDeclarations(t *testing.T) {
	// Issue #3: Panic on grouped type declarations with "duplicate node" error
	t.Parallel()

	input := `package minimal

type (
	A int
	B string
)
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source failed: %v", err)
	}

	// Should contain both type declarations
	if !hasSubstring(result, "type") {
		t.Errorf("Expected type declarations in output.\nGot:\n%s", result)
	}
}
