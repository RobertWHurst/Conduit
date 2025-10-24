package conduit

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	transport := &mockTransport{}
	encoder := &mockEncoder{}

	client := NewClient("test-service", transport, encoder)

	if client.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", client.serviceName)
	}

	if client.transport != transport {
		t.Error("Transport not set correctly")
	}

	if client.encoder != encoder {
		t.Error("Encoder not set correctly")
	}

	if client.handlerChans == nil {
		t.Error("handlerChans not initialized")
	}
}

func TestClientService(t *testing.T) {
	client := NewClient("my-service", &mockTransport{}, &mockEncoder{})

	serviceClient := client.Service("remote-service")

	if serviceClient.client != client {
		t.Error("ServiceClient not linked to client correctly")
	}

	if serviceClient.remoteServiceName != "remote-service" {
		t.Errorf("Expected remote service 'remote-service', got '%s'", serviceClient.remoteServiceName)
	}
}

func TestClientBind(t *testing.T) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	binding := client.Bind("test.event")

	if binding.client != client {
		t.Error("Binding not linked to client correctly")
	}

	if binding.eventName != "test.event" {
		t.Errorf("Expected event name 'test.event', got '%s'", binding.eventName)
	}

	if binding.handlerChan == nil {
		t.Error("Binding handler channel not initialized")
	}

	if _, ok := client.handlerChans["test.event"]; !ok {
		t.Error("Event not registered in client handler map")
	}

	if _, ok := client.handlerChans["test.event"][binding]; !ok {
		t.Error("Binding not registered in handler map")
	}
}

func TestClientClose(t *testing.T) {
	closeCalled := false
	transport := &mockTransport{
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})

	err := client.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	if !closeCalled {
		t.Error("Transport Close() not called")
	}
}

func TestClientHandleMessage(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})

	binding := client.Bind("test.event")

	msgReceived := make(chan *Message, 1)
	go func() {
		msg := binding.Next()
		msgReceived <- msg
	}()

	handler("test.event", "source-service", "reply-subject", strings.NewReader("test data"))

	select {
	case msg := <-msgReceived:
		if msg.sourceServiceName != "source-service" {
			t.Errorf("Expected source 'source-service', got '%s'", msg.sourceServiceName)
		}
		if msg.replySubject != "reply-subject" {
			t.Errorf("Expected reply subject 'reply-subject', got '%s'", msg.replySubject)
		}
	case <-time.After(time.Second):
		t.Fatal("Message not received within timeout")
	}
}

func TestClientHandleMessageMultipleBindings(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})

	binding1 := client.Bind("test.event")
	binding2 := client.Bind("test.event")

	received1 := make(chan bool, 1)
	received2 := make(chan bool, 1)

	go func() {
		binding1.Next()
		received1 <- true
	}()

	go func() {
		binding2.Next()
		received2 <- true
	}()

	handler("test.event", "source", "reply", strings.NewReader("data"))

	timeout := time.After(time.Second)
	count := 0
	for count < 2 {
		select {
		case <-received1:
			count++
		case <-received2:
			count++
		case <-timeout:
			t.Fatalf("Expected 2 bindings to receive message, got %d", count)
		}
	}
}

func TestClientHandleMessageNoBindings(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	NewClient("test-service", transport, &mockEncoder{})

	handler("nonexistent.event", "source", "reply", strings.NewReader("data"))
}

func TestClientConcurrency(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})

	var wg sync.WaitGroup
	bindingCount := 10

	for i := 0; i < bindingCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			binding := client.Bind("test.event")
			msg := binding.Next()
			if msg == nil {
				t.Error("Received nil message")
			}
			binding.Close()
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < bindingCount; i++ {
		handler("test.event", "source", "reply", strings.NewReader("data"))
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for concurrent operations")
	}
}

func TestClientMultipleServices(t *testing.T) {
	transport1 := &mockTransport{}
	transport2 := &mockTransport{}

	client1 := NewClient("service1", transport1, &mockEncoder{})
	client2 := NewClient("service2", transport2, &mockEncoder{})

	if client1.serviceName == client2.serviceName {
		t.Error("Clients should have different service names")
	}

	_ = client1
	_ = client2
}

func BenchmarkClientBind(b *testing.B) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binding := client.Bind("test.event")
		binding.Close()
	}
}

func BenchmarkClientHandleMessage(b *testing.B) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	go func() {
		for {
			msg := binding.Next()
			if msg == nil {
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler("test.event", "source", "reply", strings.NewReader("data"))
	}
}
