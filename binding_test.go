package conduit

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewBinding(t *testing.T) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})

	binding := newBinding(client, "test.event")

	if binding.client != client {
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

	client := NewClient("test-service", transport, &mockEncoder{})
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

	client := NewClient("test-service", transport, &mockEncoder{})
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

	client := NewClient("test-service", transport, &mockEncoder{})
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

func TestBindingClose(t *testing.T) {
	client := NewClient("test-service", &mockTransport{}, &mockEncoder{})
	binding := client.Bind("test.event")

	if _, ok := client.handlerChans["test.event"][binding]; !ok {
		t.Error("Binding not registered before close")
	}

	binding.Close()

	if _, ok := client.handlerChans["test.event"][binding]; ok {
		t.Error("Binding still registered after close")
	}
}

func TestBindingCloseStopsTo(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})
	binding := client.Bind("test.event")

	exited := make(chan bool, 1)

	binding.To(func(msg *Message) {})

	go func() {
		binding.To(func(msg *Message) {})
		exited <- true
	}()

	time.Sleep(50 * time.Millisecond)

	_ = handler
	binding.Close()

	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("To() goroutine did not exit after Close()")
	}
}

func TestBindingMultipleBindingsSameEvent(t *testing.T) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})

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

	client := NewClient("test-service", transport, &mockEncoder{})
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

func BenchmarkBindingNext(b *testing.B) {
	var handler func(subject, sourceServiceName, replySubject string, reader io.Reader)
	transport := &mockTransport{
		handleFunc: func(serviceName string, h func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
			handler = h
		},
	}

	client := NewClient("test-service", transport, &mockEncoder{})
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

	client := NewClient("test-service", transport, &mockEncoder{})
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
