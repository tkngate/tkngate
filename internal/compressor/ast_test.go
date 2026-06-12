package compressor

import (
	"strings"
	"testing"
)

func TestCompress(t *testing.T) {
	input := `package main

// This is a bloated function
func DoSomethingComplex() {
	// Let's do some math
	x := 10
	y := 20
	_ = x + y
}
`
	output := Compress(input)

	if strings.Contains(output, "bloated") {
		t.Errorf("Compressor failed to strip comments: %s", output)
	}

	if strings.Contains(output, "x := 10") {
		t.Errorf("Compressor failed to prune function body: %s", output)
	}

	// It should retain the function signature
	if !strings.Contains(output, "func DoSomethingComplex()") {
		t.Errorf("Compressor stripped the function signature: %s", output)
	}
}
