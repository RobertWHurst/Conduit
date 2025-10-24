package conduit

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestServiceClientSend(t *testing.T) {
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
	serviceClient := client.Service("remote-service")

	err := serviceClient.Send("test.subject", "test data")
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	if capturedServiceName != "remote-service" {
		t.Errorf("Expected service name 'remote-service', got '%s'", capturedServiceName)
	}

	if capturedSubject != "test.subject" {
		t.Errorf("Expected subject 'test.subject', got '%s'", capturedSubject)
	}

	if capturedSource != "my-service" {
		t.Errorf("Expected source 'my-service', got '%s'", capturedSource)
	}

	if capturedReplySubject != "" {
		t.Errorf("Expected empty reply subject, got '%s'", capturedReplySubject)
	}
}

func TestServiceClientRequest(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	var capturedReplySubject string

	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			capturedReplySubject = replySubject

			go func() {
				time.Sleep(50 * time.Millisecond)
				handler(replySubject, serviceName, "", strings.NewReader("response"))
			}()

			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	msg := serviceClient.Request("test.subject", "request data")

	data, err := io.ReadAll(msg)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(data) != "response" {
		t.Errorf("Expected response 'response', got '%s'", string(data))
	}

	if capturedReplySubject == "" {
		t.Error("Reply subject not set")
	}

	if len(capturedReplySubject) != 32 {
		t.Errorf("Expected reply subject length 32, got %d", len(capturedReplySubject))
	}
}

func TestServiceClientRequestWithTimeout(t *testing.T) {
	transport := &mockTransport{
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	msg := serviceClient.RequestWithTimeout("test.subject", "request", 100*time.Millisecond)

	var result string
	err := msg.Into(&result)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded error, got %v", err)
	}
}

func TestServiceClientRequestWithCtx(t *testing.T) {
	transport := &mockTransport{
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg := serviceClient.RequestWithCtx(ctx, "test.subject", "request")

	var result string
	err := msg.Into(&result)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded error, got %v", err)
	}
}

func TestServiceClientRequestWithCtxCanceled(t *testing.T) {
	transport := &mockTransport{
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := serviceClient.RequestWithCtx(ctx, "test.subject", "request")

	var result string
	err := msg.Into(&result)
	if err != context.Canceled {
		t.Errorf("Expected canceled error, got %v", err)
	}
}

func TestGenerateReplySubject(t *testing.T) {
	subject1 := generateReplySubject()
	subject2 := generateReplySubject()

	if len(subject1) != 32 {
		t.Errorf("Expected subject length 32, got %d", len(subject1))
	}

	if subject1 == subject2 {
		t.Error("Generated subjects should be unique")
	}

	for _, r := range subject1 {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !valid {
			t.Errorf("Invalid character '%c' in reply subject", r)
		}
	}
}

func TestServiceClientRequestSuccessful(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)

	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			go func() {
				time.Sleep(50 * time.Millisecond)
				handler(replySubject, serviceName, "", strings.NewReader("success"))
			}()
			return nil
		},
	}

	encoder := &mockEncoder{
		encodeFunc: func(v any) ([]byte, error) {
			return []byte("encoded"), nil
		},
		decodeFunc: func(data []byte, v any) error {
			if ptr, ok := v.(*string); ok {
				*ptr = string(data)
			}
			return nil
		},
	}

	client := NewClient("my-service", transport, encoder)
	serviceClient := client.Service("remote-service")

	msg := serviceClient.Request("test.subject", "request")

	var response string
	err := msg.Into(&response)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if response != "success" {
		t.Errorf("Expected response 'success', got '%s'", response)
	}
}

func BenchmarkServiceClientSend(b *testing.B) {
	transport := &mockTransport{
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serviceClient.Send("test.subject", "data")
	}
}

func BenchmarkServiceClientRequest(b *testing.B) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)

	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
		sendFunc: func(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
			go handler(replySubject, serviceName, "", strings.NewReader("response"))
			return nil
		},
	}

	client := NewClient("my-service", transport, &mockEncoder{})
	serviceClient := client.Service("remote-service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := serviceClient.Request("test.subject", "request")
		var result string
		msg.Into(&result)
	}
}

func BenchmarkGenerateReplySubject(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateReplySubject()
	}
}
