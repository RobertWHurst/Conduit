package protobufencoder

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestEncoderEncode(t *testing.T) {
	encoder := New()

	msg := &wrapperspb.StringValue{Value: "test"}

	encoded, err := encoder.Encode(msg)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Expected non-empty encoded data")
	}
}

func TestEncoderEncodeNonProtoMessage(t *testing.T) {
	encoder := New()

	_, err := encoder.Encode("not a proto message")
	if err == nil {
		t.Error("Expected error for non-proto message, got nil")
	}
}

func TestEncoderDecode(t *testing.T) {
	encoder := New()

	original := &wrapperspb.StringValue{Value: "test"}
	encoded, _ := encoder.Encode(original)

	result := &wrapperspb.StringValue{}
	err := encoder.Decode(encoded, result)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if result.Value != "test" {
		t.Errorf("Expected value 'test', got '%s'", result.Value)
	}
}

func TestEncoderDecodeNonProtoMessage(t *testing.T) {
	encoder := New()

	encoded := []byte{0x01, 0x02, 0x03}

	var result string
	err := encoder.Decode(encoded, &result)
	if err == nil {
		t.Error("Expected error for non-proto message, got nil")
	}
}

func TestEncoderEncodeDecodeRoundTrip(t *testing.T) {
	encoder := New()

	original := &wrapperspb.Int64Value{Value: 42}

	encoded, err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	decoded := &wrapperspb.Int64Value{}
	err = encoder.Decode(encoded, decoded)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if decoded.Value != original.Value {
		t.Errorf("Expected value %d, got %d", original.Value, decoded.Value)
	}
}

func TestEncoderDecodeInvalid(t *testing.T) {
	encoder := New()

	invalidData := []byte{0xFF, 0xFF, 0xFF}

	result := &wrapperspb.StringValue{}
	err := encoder.Decode(invalidData, result)
	if err == nil {
		t.Error("Expected error for invalid protobuf data, got nil")
	}
}

func TestEncoderCompactness(t *testing.T) {
	encoder := New()

	msg := &wrapperspb.BoolValue{Value: true}

	encoded, err := encoder.Encode(msg)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(encoded) > 10 {
		t.Errorf("Expected compact encoding (<10 bytes), got %d bytes", len(encoded))
	}
}

func BenchmarkEncoderEncode(b *testing.B) {
	encoder := New()
	msg := &wrapperspb.StringValue{Value: "benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(msg)
	}
}

func BenchmarkEncoderDecode(b *testing.B) {
	encoder := New()
	msg := &wrapperspb.StringValue{Value: "benchmark"}
	encoded, _ := proto.Marshal(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &wrapperspb.StringValue{}
		encoder.Decode(encoded, result)
	}
}
