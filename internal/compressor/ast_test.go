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

func TestCompress_JSON(t *testing.T) {
	input := `{
  "tool_calls": [
    {
      "id": "call_123",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\":\"San Francisco\"}"
      }
    }
  ]
}`
	output := Compress(input)
	if output != input {
		t.Errorf("Compressor modified JSON payload. Expected %q, got %q", input, output)
	}
}

func TestCompressJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Basic Function",
			input: `function foo(x: string) {
	console.log(x);
	return x;
}`,
			expected: `function foo(x: string) { /* body omitted */ }`,
		},
		{
			name: "Arrow Function",
			input: `const a = () => {
	// do something
	return 1;
}`,
			expected: `const a = () => { /* body omitted */ }`,
		},
		{
			name: "Class with Methods",
			input: `export class MyClass<T> {
	constructor(public arg: T) {
		this.init();
	}
	public doAction(): void {
		console.log("action");
	}
}`,
			expected: `export class MyClass<T> {
	constructor(public arg: T) { /* body omitted */ }
	public doAction(): void { /* body omitted */ }
}`,
		},
		{
			name: "Control Flow Preserved",
			input: `if (true) {
	doX();
}
for (let i=0; i<10; i++) {
	console.log(i);
}`,
			expected: `if (true) {
	doX();
}
for (let i=0; i<10; i++) {
	console.log(i);
}`,
		},
		{
			name: "Object Literal Preserved but Methods Dropped",
			input: `const obj = {
	a: 1,
	b: () => {
		return 2;
	},
	c() {
		return 3;
	}
}`,
			expected: `const obj = {
	a: 1,
	b: () => { /* body omitted */ },
	c() { /* body omitted */ }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressJS(tt.input)
			// Ignore whitespace differences in testing
			gotCompact := strings.Join(strings.Fields(got), " ")
			expCompact := strings.Join(strings.Fields(tt.expected), " ")
			if gotCompact != expCompact {
				t.Errorf("\nGot:\n%s\nExpected:\n%s", got, tt.expected)
			}
		})
	}
}

func TestHeuristics(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		isJSON   bool
		isGo     bool
		isPython bool
		isJS     bool
	}{
		{
			name: "Go Code",
			content: `package main
			import "fmt"
			func main() {
				fmt.Println("hello")
			}
			func help() {}`,
			isJSON:   false,
			isGo:     true,
			isPython: false,
			isJS:     false,
		},
		{
			name: "Python Code",
			content: `def foo():
				pass
			class Bar:
				def baz(self):
					pass`,
			isJSON:   false,
			isGo:     false,
			isPython: true,
			isJS:     false,
		},
		{
			name: "JS Code",
			content: `function a() {}
			const b = () => {}
			export class C {}`,
			isJSON:   false,
			isGo:     false,
			isPython: false,
			isJS:     true,
		},
		{
			name: "JSON Payload",
			content: `{"key": "value", "list": [1,2,3]}`,
			isJSON:   true,
			isGo:     false,
			isPython: false,
			isJS:     false,
		},
		{
			name: "Natural Language",
			content: `This is just a regular text sentence. It should not be compressed.`,
			isJSON:   false,
			isGo:     false,
			isPython: false,
			isJS:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJSON(tt.content); got != tt.isJSON {
				t.Errorf("isJSON() = %v, want %v", got, tt.isJSON)
			}
			if got := isGoCode(tt.content); got != tt.isGo {
				t.Errorf("isGoCode() = %v, want %v", got, tt.isGo)
			}
			if got := isPythonCode(tt.content); got != tt.isPython {
				t.Errorf("isPythonCode() = %v, want %v", got, tt.isPython)
			}
			if got := isJSCode(tt.content); got != tt.isJS {
				t.Errorf("isJSCode() = %v, want %v", got, tt.isJS)
			}
		})
	}
}
