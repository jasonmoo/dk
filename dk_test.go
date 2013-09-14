package main

import (
	"testing"
)

func TestExample(t *testing.T) {
	if false {
		t.Errorf("%s", "hi")
	}
}

func BenchmarkExample(b *testing.B) {
	a, aa := 1, 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a, aa = aa, a
	}
}
