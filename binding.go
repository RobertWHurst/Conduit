package conduit

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

// Binding represents a subscription to messages on a specific subject.
// Bindings provide two ways to consume messages: Next() for blocking retrieval
// and To() for handler-based processing.
type Binding struct {
	client      *Client
	bindType    BindType
	eventName   string
	handlerChan chan *Message
}

func newBinding(client *Client, bindType BindType, eventName string) *Binding {
	b := &Binding{
		client:      client,
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

// Next blocks until the next message arrives and returns it.
// This is useful for processing messages sequentially in a loop.
func (b *Binding) Next() *Message {
	msg := <-b.handlerChan
	if b.bindType == BindTypeOnce {
		defer b.Unbind()
	}
	return msg
}

// To spawns a goroutine that calls the handler for each message.
// The handler runs asynchronously and continues until the binding is closed.
func (b *Binding) To(handler func(msg *Message)) *Binding {
	go func() {
		for {
			msg, ok := <-b.handlerChan
			if b.bindType == BindTypeOnce {
				defer b.Unbind()
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
	if b.bindType == BindTypeQueue {
		b.client.queueHandlerChansMu.Lock()
		defer b.client.queueHandlerChansMu.Unlock()
		delete(b.client.queueHandlerChans[b.eventName], b)
	} else {
		b.client.handlerChansMu.Lock()
		defer b.client.handlerChansMu.Unlock()
		delete(b.client.handlerChans[b.eventName], b)
	}
	close(b.handlerChan)
}
