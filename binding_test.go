package conduit

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewBinding(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})

	binding := newBinding(client, BindTypeNormal, "test.event")

	if binding.conduit != client {
		t.Error("Binding client not set correctly")
	}

	if binding.eventName != "test.event" {
		t.Errorf("Expected event name 'test.event', got '%s'", binding.eventName)
	}

	if binding.handlerChan == nil {
		t.Error("Handler channel not initialized")
	}

	if cap(binding.handlerChan) != 100 {
		t.Errorf("Expected handler channel capacity 100, got %d", cap(binding.handlerChan))
	}

	if _, ok := client.handlerChans["test.event"][binding]; !ok {
		t.Error("Binding not registered in client handler map")
	}
}

func TestBindingNext(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	testData := "test message"

	go func() {
		time.Sleep(50 * time.Millisecond)
		handler("test.event", "source", "reply", strings.NewReader(testData))
	}()

	msg := binding.Next()

	if msg.sourceServiceName != "source" {
		t.Errorf("Expected source 'source', got '%s'", msg.sourceServiceName)
	}

	if msg.replySubject != "reply" {
		t.Errorf("Expected reply subject 'reply', got '%s'", msg.replySubject)
	}

	data, _ := io.ReadAll(msg.data)
	if string(data) != testData {
		t.Errorf("Expected data '%s', got '%s'", testData, string(data))
	}
}

func TestBindingTo(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	received := make(chan *Message, 1)

	binding.To(func(msg *Message) {
		received <- msg
	})

	time.Sleep(50 * time.Millisecond)

	handler("test.event", "source", "reply", strings.NewReader("test"))

	select {
	case msg := <-received:
		if msg.sourceServiceName != "source" {
			t.Errorf("Expected source 'source', got '%s'", msg.sourceServiceName)
		}
	case <-time.After(time.Second):
		t.Fatal("Handler not called within timeout")
	}
}

func TestBindingToMultipleMessages(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	count := 0
	var mu sync.Mutex

	binding.To(func(msg *Message) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 5; i++ {
		handler("test.event", "source", "reply", strings.NewReader("test"))
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	finalCount := count
	mu.Unlock()

	if finalCount != 5 {
		t.Errorf("Expected 5 messages, got %d", finalCount)
	}
}

func TestBindingUnbind(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	if _, ok := client.handlerChans["test.event"][binding]; !ok {
		t.Error("Binding not registered before unbind")
	}

	binding.Unbind()

	if _, ok := client.handlerChans["test.event"][binding]; ok {
		t.Error("Binding still registered after unbind")
	}
}

func TestBindingUnbindStopsTo(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	exited := make(chan bool, 1)

	binding.To(func(msg *Message) {})

	go func() {
		binding.To(func(msg *Message) {})
		exited <- true
	}()

	time.Sleep(50 * time.Millisecond)

	_ = handler
	binding.Unbind()

	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("To() goroutine did not exit after Unbind()")
	}
}

func TestBindingMultipleBindingsSameEvent(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})

	binding1 := client.Bind("test.event")
	binding2 := client.Bind("test.event")

	count1 := 0
	count2 := 0
	var mu sync.Mutex

	binding1.To(func(msg *Message) {
		mu.Lock()
		count1++
		mu.Unlock()
	})

	binding2.To(func(msg *Message) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	time.Sleep(50 * time.Millisecond)

	handler("test.event", "source", "reply", strings.NewReader("test"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	finalCount1 := count1
	finalCount2 := count2
	mu.Unlock()

	if finalCount1 != 1 {
		t.Errorf("Expected binding1 to receive 1 message, got %d", finalCount1)
	}

	if finalCount2 != 1 {
		t.Errorf("Expected binding2 to receive 1 message, got %d", finalCount2)
	}
}

func TestBindingChannelBuffer(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	for i := 0; i < 100; i++ {
		handler("test.event", "source", "reply", strings.NewReader("test"))
	}

	count := 0
	for i := 0; i < 100; i++ {
		select {
		case <-binding.handlerChan:
			count++
		case <-time.After(10 * time.Millisecond):
			break
		}
	}

	if count != 100 {
		t.Errorf("Expected 100 buffered messages, got %d", count)
	}
}

func TestBindingIsBound(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	if !binding.IsBound() {
		t.Error("Expected binding to be bound after creation")
	}

	binding.Unbind()

	if binding.IsBound() {
		t.Error("Expected binding to not be bound after unbind")
	}
}

func TestBindingNextAfterUnbind(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	binding.Unbind()

	msg := binding.Next()
	if msg.err != ErrBindingClosed {
		t.Errorf("Expected ErrBindingClosed, got %v", msg.err)
	}
}

func TestBindingToAfterUnbind(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	binding.Unbind()

	called := false
	returnedBinding := binding.To(func(msg *Message) {
		called = true
	})

	if returnedBinding != binding {
		t.Error("Expected To() to return the same binding")
	}

	time.Sleep(100 * time.Millisecond)

	if called {
		t.Error("Handler should not be called on closed binding")
	}
}

func TestBindingUnbindIdempotent(t *testing.T) {
	client := New("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	// Should not panic when calling Unbind multiple times
	binding.Unbind()
	binding.Unbind()
	binding.Unbind()

	if binding.IsBound() {
		t.Error("Expected binding to remain unbound")
	}
}

func TestBindingTypeOnceWithNext(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := newBinding(client, BindTypeOnce, "test.event")

	if !binding.IsBound() {
		t.Error("Expected binding to be bound initially")
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		handler("test.event", "source", "reply", strings.NewReader("test"))
	}()

	msg := binding.Next()

	if msg.sourceServiceName != "source" {
		t.Errorf("Expected source 'source', got '%s'", msg.sourceServiceName)
	}

	// Binding should auto-unbind after one message
	time.Sleep(50 * time.Millisecond)
	if binding.IsBound() {
		t.Error("Expected BindTypeOnce to auto-unbind after Next()")
	}
}

func TestBindingTypeOnceWithTo(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := newBinding(client, BindTypeOnce, "test.event")

	received := make(chan bool, 1)
	binding.To(func(msg *Message) {
		received <- true
	})

	time.Sleep(50 * time.Millisecond)

	handler("test.event", "source", "reply", strings.NewReader("test"))

	select {
	case <-received:
		// Good, handler was called
	case <-time.After(time.Second):
		t.Fatal("Handler not called within timeout")
	}

	// Wait a bit for auto-unbind to happen
	time.Sleep(100 * time.Millisecond)

	if binding.IsBound() {
		t.Error("Expected BindTypeOnce to auto-unbind after To() handler")
	}

	// Send another message - should not be received
	handler("test.event", "source", "reply", strings.NewReader("test2"))

	select {
	case <-received:
		t.Error("Handler should not be called after auto-unbind")
	case <-time.After(100 * time.Millisecond):
		// Good, handler not called
	}
}

func BenchmarkBindingNext(b *testing.B) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	go func() {
		for i := 0; i < b.N; i++ {
			handler("test.event", "source", "reply", strings.NewReader("test"))
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binding.Next()
	}
}

func BenchmarkBindingTo(b *testing.B) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := New("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	var wg sync.WaitGroup
	wg.Add(b.N)

	binding.To(func(msg *Message) {
		wg.Done()
	})

	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler("test.event", "source", "reply", strings.NewReader("test"))
	}

	wg.Wait()
}
