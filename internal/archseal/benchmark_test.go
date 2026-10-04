package archseal

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkExtractImports(b *testing.B) {
	var source strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&source, "import { x%d } from \"../layer/file%d\";\n", i, i)
	}
	input := source.String()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		refs := extractImports(input)
		if len(refs) != 2000 {
			b.Fatalf("expected 2000 imports, got %d", len(refs))
		}
	}
}
