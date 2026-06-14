package tokenizer

import (
	"testing"
)

func TestCounter_Count_NaturalLanguage(t *testing.T) {
	c, _ := NewCounter()
	text := "This is a simple natural language test sentence with no special characters."
	
	count := c.Count(text, "gpt-4")
	if count <= 1 {
		t.Errorf("Expected count > 1, got %d", count)
	}
	
	// Length is ~75 chars, expected ~18 tokens
	if count < 10 || count > 25 {
		t.Errorf("Count %d is wildly off for natural language", count)
	}
}

func TestCounter_Count_Code(t *testing.T) {
	c, _ := NewCounter()
	text := `{"key": "value", "items": [1, 2, 3], "nested": {"a": "b"}}`
	
	count := c.Count(text, "gpt-4")
	
	// Should be treated as code (~3.2 chars per token)
	if count < 15 || count > 25 {
		t.Errorf("Count %d is off for JSON payload", count)
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		provider     string
		model        string
		input        int
		output       int
		expectedCost float64
	}{
		{"openai", "gpt-6o", 1000, 1000, 0.020},        // 0.005 + 0.015
		{"anthropic", "claude-4.5-sonnet", 1000, 1000, 0.018}, // 0.003 + 0.015
		{"deepseek", "deepseek-chat-v3", 10000, 10000, 0.0042}, // 0.0014 + 0.0028
	}

	for _, tt := range tests {
		cost := EstimateCost(tt.provider, tt.model, tt.input, tt.output)
		// Float comparison with tolerance
		diff := cost - tt.expectedCost
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("EstimateCost(%s, %s) = %f, want %f", tt.provider, tt.model, cost, tt.expectedCost)
		}
	}
}
