package natstransport

import (
	"testing"
)

func TestNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "single value",
			input:    []string{"service"},
			expected: "conduit.service",
		},
		{
			name:     "multiple values",
			input:    []string{"user", "service"},
			expected: "conduit.user.service",
		},
		{
			name:     "empty values filtered",
			input:    []string{"user", "", "service"},
			expected: "conduit.user.service",
		},
		{
			name:     "camel case conversion",
			input:    []string{"userService"},
			expected: "conduit.user-service",
		},
		{
			name:     "underscore conversion",
			input:    []string{"user_service"},
			expected: "conduit.user-service",
		},
		{
			name:     "mixed case",
			input:    []string{"UserService"},
			expected: "conduit.User-service",
		},
		{
			name:     "dots preserved",
			input:    []string{"user.service"},
			expected: "conduit.user.service",
		},
		{
			name:     "wildcards preserved",
			input:    []string{"user.*"},
			expected: "conduit.user.*",
		},
		{
			name:     "numbers preserved",
			input:    []string{"service123"},
			expected: "conduit.service123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := namespace(tt.input...)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatForNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "test",
			expected: "test",
		},
		{
			name:     "uppercase",
			input:    "TEST",
			expected: "TEST",
		},
		{
			name:     "camel case",
			input:    "testValue",
			expected: "test-value",
		},
		{
			name:     "pascal case",
			input:    "TestValue",
			expected: "Test-value",
		},
		{
			name:     "multiple capitals",
			input:    "HTTPServer",
			expected: "HTTPServer",
		},
		{
			name:     "underscore to dash",
			input:    "test_value",
			expected: "test-value",
		},
		{
			name:     "dash preserved",
			input:    "test-value",
			expected: "test-value",
		},
		{
			name:     "dot preserved",
			input:    "test.value",
			expected: "test.value",
		},
		{
			name:     "wildcard preserved",
			input:    "test*",
			expected: "test*",
		},
		{
			name:     "numbers preserved",
			input:    "test123",
			expected: "test123",
		},
		{
			name:     "mixed numbers and letters",
			input:    "test123Value",
			expected: "test123Value",
		},
		{
			name:     "special characters removed",
			input:    "test@value",
			expected: "testvalue",
		},
		{
			name:     "spaces removed",
			input:    "test value",
			expected: "testvalue",
		},
		{
			name:     "multiple underscores",
			input:    "test__value",
			expected: "test--value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatForNamespace(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNamespaceWithComplexValues(t *testing.T) {
	result := namespace("myUserService", "getUserById", "request")
	expected := "conduit.my-user-service.get-user-by-id.request"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatForNamespacePerformance(t *testing.T) {
	input := "veryLongServiceNameWithManyCapitalLetters"

	result := formatForNamespace(input)

	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

func BenchmarkNamespace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		namespace("userService", "getUserById")
	}
}

func BenchmarkFormatForNamespace(b *testing.B) {
	input := "userServiceName"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatForNamespace(input)
	}
}

func BenchmarkFormatForNamespaceLong(b *testing.B) {
	input := "veryLongServiceNameWithManyCapitalLettersAndNumbers123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatForNamespace(input)
	}
}
