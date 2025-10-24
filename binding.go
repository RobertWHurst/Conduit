package conduit

type Binding struct {
	client      *Client
	eventName   string
	handlerChan chan *Message
}

func newBinding(client *Client, eventName string) *Binding {
	b := &Binding{
		client:      client,
		eventName:   eventName,
		handlerChan: make(chan *Message, 100),
	}
	client.handlerChansMu.Lock()
	defer client.handlerChansMu.Unlock()
	if _, ok := client.handlerChans[eventName]; !ok {
		client.handlerChans[eventName] = make(map[*Binding]chan *Message)
	}
	client.handlerChans[eventName][b] = b.handlerChan
	return b
}

func (b *Binding) Next() *Message {
	return <-b.handlerChan
}

func (b *Binding) To(handler func(msg *Message)) {
	go func() {
		for {
			msg, ok := <-b.handlerChan
			if !ok {
				break
			}
			handler(msg)
		}
	}()
}

func (b *Binding) Close() {
	b.client.handlerChansMu.Lock()
	defer b.client.handlerChansMu.Unlock()
	delete(b.client.handlerChans[b.eventName], b)
	close(b.handlerChan)
}
