package reorder_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/toejough/go-reorder"
)

// generateSource creates Go source with the specified number of declarations.
func generateSource(numTypes, numFuncs, numConsts int) string {
	var sb strings.Builder
	sb.WriteString("package benchmark\n\n")
	sb.WriteString("import \"fmt\"\n\n")

	// Generate constants
	for i := 0; i < numConsts; i++ {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("const Const%d = %d\n", i, i))
		} else {
			sb.WriteString(fmt.Sprintf("const const%d = %d\n", i, i))
		}
	}
	sb.WriteString("\n")

	// Generate types with constructors and methods
	for i := 0; i < numTypes; i++ {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("type Type%d struct { value int }\n\n", i))
			sb.WriteString(fmt.Sprintf("func NewType%d() *Type%d { return &Type%d{} }\n\n", i, i, i))
			sb.WriteString(fmt.Sprintf("func (t *Type%d) Method() { fmt.Println(t.value) }\n\n", i))
		} else {
			sb.WriteString(fmt.Sprintf("type type%d struct { value int }\n\n", i))
			sb.WriteString(fmt.Sprintf("func newType%d() *type%d { return &type%d{} }\n\n", i, i, i))
			sb.WriteString(fmt.Sprintf("func (t *type%d) method() { fmt.Println(t.value) }\n\n", i))
		}
	}

	// Generate standalone functions
	for i := 0; i < numFuncs; i++ {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("func Func%d() { fmt.Println(%d) }\n\n", i, i))
		} else {
			sb.WriteString(fmt.Sprintf("func func%d() { fmt.Println(%d) }\n\n", i, i))
		}
	}

	return sb.String()
}

// BenchmarkSource_Small benchmarks Source with ~100 line file.
func BenchmarkSource_Small(b *testing.B) {
	src := generateSource(5, 5, 10) // ~100 lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.Source(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSource_Medium benchmarks Source with ~1000 line file.
func BenchmarkSource_Medium(b *testing.B) {
	src := generateSource(50, 50, 100) // ~1000 lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.Source(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSource_Large benchmarks Source with ~10000 line file.
func BenchmarkSource_Large(b *testing.B) {
	src := generateSource(500, 500, 1000) // ~10000 lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.Source(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSourceWithConfig_Small benchmarks SourceWithConfig with ~100 line file.
func BenchmarkSourceWithConfig_Small(b *testing.B) {
	src := generateSource(5, 5, 10)
	cfg := reorder.DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.SourceWithConfig(src, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSourceWithConfig_Medium benchmarks SourceWithConfig with ~1000 line file.
func BenchmarkSourceWithConfig_Medium(b *testing.B) {
	src := generateSource(50, 50, 100)
	cfg := reorder.DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.SourceWithConfig(src, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSourceWithConfig_Large benchmarks SourceWithConfig with ~10000 line file.
func BenchmarkSourceWithConfig_Large(b *testing.B) {
	src := generateSource(500, 500, 1000)
	cfg := reorder.DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.SourceWithConfig(src, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAnalyzeSectionOrder benchmarks the analysis function.
func BenchmarkAnalyzeSectionOrder_Medium(b *testing.B) {
	src := generateSource(50, 50, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reorder.AnalyzeSectionOrder(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}
