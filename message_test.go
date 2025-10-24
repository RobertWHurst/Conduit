package conduit

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type mockEncoder struct {
	encodeFunc func(v any) ([]byte, error)
	decodeFunc func(data []byte, v any) error
}

func (m *mockEncoder) Encode(v any) ([]byte, error) {
	if m.encodeFunc != nil {
		return m.encodeFunc(v)
	}
	return []byte("encoded"), nil
}

func (m *mockEncoder) Decode(data []byte, v any) error {
	if m.decodeFunc != nil {
		return m.decodeFunc(data, v)
	}
	return nil
}

type mockTransport struct {
	sendFunc        func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error
	handleFunc      func(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))
	handleQueueFunc func(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))
	closeFunc       func() error
}

func (m *mockTransport) Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
	if m.sendFunc != nil {
		return m.sendFunc(serviceName, subject, sourceServiceName, replySubject, reader)
	}
	return nil
}

func (m *mockTransport) Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
	if m.handleFunc != nil {
		m.handleFunc(serviceName, handler)
	}
}

func (m *mockTransport) HandleQueue(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
	if m.handleQueueFunc != nil {
		m.handleQueueFunc(serviceName, handler)
	}
}

func (m *mockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestMessageInto(t *testing.T) {
	encoder := &mockEncoder{
		decodeFunc: func(data []byte, v any) error {
			if ptr, ok := v.(*string); ok {
				*ptr = "decoded"
			}
			return nil
		},
	}

	client := NewClient("test-service", &mockTransport{}, encoder)

	msg := &Message{
		sourceServiceName: "source",
		replySubject:      "reply",
		data:              strings.NewReader("test data"),
		client:            client,
	}

	var result string
	err := msg.Into(&result)
	if err != nil {
		t.Fatalf("Into() failed: %v", err)
	}

	if result != "decoded" {
		t.Errorf("Expected 'decoded', got '%s'", result)
	}
}

func TestMessageIntoWithError(t *testing.T) {
	expectedErr := errors.New("decode error")
	encoder := &mockEncoder{
		decodeFunc: func(data []byte, v any) error {
			return expectedErr
		},
	}

	client := NewClient("test-service", &mockTransport{}, encoder)

	msg := &Message{
		data:   strings.NewReader("test data"),
		client: client,
	}

	var result string
	err := msg.Into(&result)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMessageIntoWithPriorError(t *testing.T) {
	expectedErr := errors.New("prior error")
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	msg := &Message{
		data:   strings.NewReader("test data"),
		client: client,
		err:    expectedErr,
	}

	var result string
	err := msg.Into(&result)
	if err != expectedErr {
		t.Errorf("Expected prior error %v, got %v", expectedErr, err)
	}
}

func TestMessageRead(t *testing.T) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	testData := "test data"
	msg := &Message{
		data:   strings.NewReader(testData),
		client: client,
	}

	buf := make([]byte, len(testData))
	n, err := msg.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to read %d bytes, got %d", len(testData), n)
	}

	if string(buf) != testData {
		t.Errorf("Expected '%s', got '%s'", testData, string(buf))
	}
}

func TestMessageReadWithError(t *testing.T) {
	expectedErr := errors.New("prior error")
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	msg := &Message{
		data:   strings.NewReader("test"),
		client: client,
		err:    expectedErr,
	}

	buf := make([]byte, 10)
	_, err := msg.Read(buf)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMessageReply(t *testing.T) {
	var capturedServiceName string
	var capturedSubject string
	var capturedSource string
	var capturedReplySubject string

	transport := &mockTransport{
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			capturedServiceName = serviceName
			capturedSubject = subject
			capturedSource = sourceServiceName
			capturedReplySubject = replySubject
			return nil
		},
	}

	encoder := &mockEncoder{
		encodeFunc: func(v any) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}

	client := NewClient("my-service", transport, encoder)

	msg := &Message{
		sourceServiceName: "remote-service",
		replySubject:      "reply-subject",
		data:              strings.NewReader(""),
		client:            client,
	}

	err := msg.Reply("response")
	if err != nil {
		t.Fatalf("Reply() failed: %v", err)
	}

	if capturedServiceName != "remote-service" {
		t.Errorf("Expected service name 'remote-service', got '%s'", capturedServiceName)
	}

	if capturedSubject != "reply-subject" {
		t.Errorf("Expected subject 'reply-subject', got '%s'", capturedSubject)
	}

	if capturedSource != "my-service" {
		t.Errorf("Expected source 'my-service', got '%s'", capturedSource)
	}

	if capturedReplySubject != "" {
		t.Errorf("Expected empty reply subject, got '%s'", capturedReplySubject)
	}
}

func TestMessageReplyWithError(t *testing.T) {
	expectedErr := errors.New("prior error")
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	msg := &Message{
		client: client,
		err:    expectedErr,
	}

	err := msg.Reply("response")
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestIntoDataReader(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "io.Reader",
			input:    strings.NewReader("test"),
			expected: "test",
		},
		{
			name:     "byte slice",
			input:    []byte("test"),
			expected: "test",
		},
		{
			name:     "string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "struct",
			input:    struct{ Name string }{Name: "test"},
			expected: "encoded",
		},
	}

	encoder := &mockEncoder{
		encodeFunc: func(v any) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := intoDataReader(encoder, tt.input)
			if err != nil {
				t.Fatalf("intoDataReader() failed: %v", err)
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(data))
			}
		})
	}
}

func TestIntoDataReaderWithEncodeError(t *testing.T) {
	expectedErr := errors.New("encode error")
	encoder := &mockEncoder{
		encodeFunc: func(v any) ([]byte, error) {
			return nil, expectedErr
		},
	}

	_, err := intoDataReader(encoder, struct{}{})
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMaxDecodeSize(t *testing.T) {
	oldMax := MaxDecodeSize
	defer func() { MaxDecodeSize = oldMax }()

	MaxDecodeSize = 10

	encoder := &mockEncoder{
		decodeFunc: func(data []byte, v any) error {
			if ptr, ok := v.(*string); ok {
				*ptr = string(data)
			}
			return nil
		},
	}

	client := NewClient("test-service", &mockTransport{}, encoder)

	largeData := strings.Repeat("a", 20)
	msg := &Message{
		data:   strings.NewReader(largeData),
		client: client,
	}

	var result string
	err := msg.Into(&result)
	if err != nil {
		t.Fatalf("Into() failed: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("Expected 10 bytes (limited by MaxDecodeSize), got %d", len(result))
	}
}

func BenchmarkMessageInto(b *testing.B) {
	encoder := &mockEncoder{
		decodeFunc: func(data []byte, v any) error {
			return nil
		},
	}

	client := NewClient("test-service", &mockTransport{}, encoder)

	data := bytes.Repeat([]byte("a"), 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &Message{
			data:   bytes.NewReader(data),
			client: client,
		}
		var result string
		msg.Into(&result)
	}
}

func BenchmarkMessageRead(b *testing.B) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	data := bytes.Repeat([]byte("a"), 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &Message{
			data:   bytes.NewReader(data),
			client: client,
		}
		buf := make([]byte, 1024)
		msg.Read(buf)
	}
}
