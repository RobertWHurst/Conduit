package conduit

import (
	"context"
	"errors"
	"sync"
)

// BindType specifies whether a binding receives all messages (broadcast)
// or only a share of messages (load-balanced).
type BindType int

const (
	// BindTypeNormal means all instances receive each message (fan-out/broadcast).
	BindTypeNormal BindType = iota
	// BindTypeOnce is like BindTypeNormal but will auto-unbind after one message.
	BindTypeOnce
	// BindTypeQueue means only one instance receives each message (load-balanced).
	BindTypeQueue
)

var ErrBindingClosed = errors.New("binding is closed")

// Binding represents a subscription to messages on a specific subject.
// Bindings provide two ways to consume messages: Next() for blocking retrieval
// and To() for handler-based processing.
type Binding struct {
	mu          sync.Mutex
	conduit     *Conduit
	bindType    BindType
	eventName   string
	handlerChan chan *Message

	ctxCancel context.CancelFunc
	ctx       context.Context
}

func newBinding(client *Conduit, bindType BindType, eventName string) *Binding {
	b := &Binding{
		conduit:     client,
		bindType:    bindType,
		eventName:   eventName,
		handlerChan: make(chan *Message, 100),
	}

	if bindType == BindTypeQueue {
		client.queueHandlerChansMu.Lock()
		defer client.queueHandlerChansMu.Unlock()
		if _, ok := client.queueHandlerChans[eventName]; !ok {
			client.queueHandlerChans[eventName] = make(map[*Binding]chan *Message)
		}
		client.queueHandlerChans[eventName][b] = b.handlerChan
	} else {
		client.handlerChansMu.Lock()
		defer client.handlerChansMu.Unlock()
		if _, ok := client.handlerChans[eventName]; !ok {
			client.handlerChans[eventName] = make(map[*Binding]chan *Message)
		}
		client.handlerChans[eventName][b] = b.handlerChan
	}

	return b
}

func (b *Binding) WithCtx(ctx context.Context) *Binding {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctxCancel != nil {
		panic("WithCtx can only be called once per Binding")
	}

	ctx, cancel := context.WithCancel(ctx)

	b.ctxCancel = cancel
	b.ctx = ctx

	go func() {
		<-ctx.Done()
		b.Unbind()
	}()

	return b
}

// IsBound returns true if the binding is currently active.
func (b *Binding) IsBound() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.handlerChan != nil
}

// Next blocks until the next message arrives and returns it.
// This is useful for processing messages sequentially in a loop.
func (b *Binding) Next() *Message {
	b.mu.Lock()
	handlerChan := b.handlerChan
	b.mu.Unlock()

	if handlerChan == nil {
		return &Message{err: ErrBindingClosed}
	}

	msg := <-handlerChan
	if b.bindType == BindTypeOnce {
		b.Unbind()
	}
	return msg
}

// To spawns a goroutine that calls the handler for each message.
// The handler runs asynchronously and continues until the binding is closed.
func (b *Binding) To(handler func(msg *Message)) *Binding {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.handlerChan == nil {
		return b
	}

	go func() {
		for b.IsBound() {
			b.mu.Lock()
			handlerChan := b.handlerChan
			b.mu.Unlock()

			msg, ok := <-handlerChan
			if b.bindType == BindTypeOnce {
				b.Unbind()
			}
			if !ok {
				break
			}
			handler(msg)
		}
	}()

	return b
}

// Unbind unsubscribes from messages and frees resources.
// Any goroutines spawned by To() will exit after Close is called.
func (b *Binding) Unbind() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlerChan == nil {
		return
	}

	if b.ctxCancel != nil {
		b.ctxCancel()
	}

	if b.bindType == BindTypeQueue {
		b.conduit.queueHandlerChansMu.Lock()
		defer b.conduit.queueHandlerChansMu.Unlock()
		delete(b.conduit.queueHandlerChans[b.eventName], b)
	} else {
		b.conduit.handlerChansMu.Lock()
		defer b.conduit.handlerChansMu.Unlock()
		delete(b.conduit.handlerChans[b.eventName], b)
	}
	close(b.handlerChan)
	b.handlerChan = nil
}
