package msgpack

import (
	"testing"
)

type testStruct struct {
	Name  string `msgpack:"name"`
	Value int    `msgpack:"value"`
}

func TestEncoderEncode(t *testing.T) {
	encoder := New()

	data := testStruct{Name: "test", Value: 42}

	encoded, err := encoder.Encode(data)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Expected non-empty encoded data")
	}
}

func TestEncoderDecode(t *testing.T) {
	encoder := New()

	original := testStruct{Name: "test", Value: 42}
	encoded, _ := encoder.Encode(original)

	var result testStruct
	err := encoder.Decode(encoded, &result)
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

func TestEncoderDecodeInvalid(t *testing.T) {
	encoder := New()

	invalidData := []byte{0xFF, 0xFF, 0xFF}

	var result testStruct
	err := encoder.Decode(invalidData, &result)
	if err == nil {
		t.Error("Expected error for invalid msgpack data, got nil")
	}
}

func TestEncoderEncodeNil(t *testing.T) {
	encoder := New()

	encoded, err := encoder.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Expected non-empty encoded data for nil")
	}
}

func TestEncoderCompactness(t *testing.T) {
	encoder := New()

	data := testStruct{Name: "compact", Value: 100}

	encoded, err := encoder.Encode(data)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(encoded) > 30 {
		t.Errorf("Expected compact encoding (<30 bytes), got %d bytes", len(encoded))
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
	data := testStruct{Name: "benchmark", Value: 999}
	encoded, _ := encoder.Encode(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result testStruct
		encoder.Decode(encoded, &result)
	}
}
