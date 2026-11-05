package generator

import (
	"testing"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func BenchmarkGeneratorGenerate(b *testing.B) {
	const (
		alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		size     = 8
	)
	gen, err := New(alphabet, size)
	if err != nil {
		b.Fatalf("initialize generator failed: %v", err)
	}

	b.Run("my-generator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := gen.Generate()
			if err != nil {
				b.Fatalf("failed to generate: %v", err)
			}
		}
	})

	b.Run("gonanoid-generator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := gonanoid.Generate(alphabet, size)
			if err != nil {
				b.Fatalf("failed to generate: %v", err)
			}
		}
	})

}
