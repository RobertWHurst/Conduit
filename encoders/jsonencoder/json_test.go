package jsonencoder

import (
	"testing"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestEncoderEncode(t *testing.T) {
	encoder := New()

	data := testStruct{Name: "test", Value: 42}

	encoded, err := encoder.Encode(data)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	expected := `{"name":"test","value":42}`
	if string(encoded) != expected {
		t.Errorf("Expected %s, got %s", expected, string(encoded))
	}
}

func TestEncoderDecode(t *testing.T) {
	encoder := New()

	data := []byte(`{"name":"test","value":42}`)

	var result testStruct
	err := encoder.Decode(data, &result)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", result.Name)
	}

	if result.Value != 42 {
		t.Errorf("Expected value 42, got %d", result.Value)
	}
}

func TestEncoderEncodeDecodeRoundTrip(t *testing.T) {
	encoder := New()

	original := testStruct{Name: "roundtrip", Value: 123}

	encoded, err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	var decoded testStruct
	err = encoder.Decode(encoded, &decoded)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Expected name '%s', got '%s'", original.Name, decoded.Name)
	}

	if decoded.Value != original.Value {
		t.Errorf("Expected value %d, got %d", original.Value, decoded.Value)
	}
}

func TestEncoderDecodeInvalidJSON(t *testing.T) {
	encoder := New()

	invalidData := []byte(`{invalid json}`)

	var result testStruct
	err := encoder.Decode(invalidData, &result)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestEncoderEncodeNil(t *testing.T) {
	encoder := New()

	encoded, err := encoder.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) failed: %v", err)
	}

	if string(encoded) != "null" {
		t.Errorf("Expected 'null', got '%s'", string(encoded))
	}
}

func TestEncoderDecodeEmpty(t *testing.T) {
	encoder := New()

	var result testStruct
	err := encoder.Decode([]byte("{}"), &result)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if result.Name != "" {
		t.Errorf("Expected empty name, got '%s'", result.Name)
	}

	if result.Value != 0 {
		t.Errorf("Expected zero value, got %d", result.Value)
	}
}

func BenchmarkEncoderEncode(b *testing.B) {
	encoder := New()
	data := testStruct{Name: "benchmark", Value: 999}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(data)
	}
}

func BenchmarkEncoderDecode(b *testing.B) {
	encoder := New()
	data := []byte(`{"name":"benchmark","value":999}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result testStruct
		encoder.Decode(data, &result)
	}
}
