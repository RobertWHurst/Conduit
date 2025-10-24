package conduit

import (
	"io"
	"sync"
)

var MaxDecodeSize = int64(1024 * 1024 * 5) // 5 MB

type Client struct {
	serviceName    string
	transport      Transport
	encoder        Encoder
	handlerChansMu sync.RWMutex
	handlerChans   map[string]map[*Binding]chan *Message
}

func NewClient(serviceName string, transport Transport, encoder Encoder) *Client {
	c := &Client{
		serviceName:  serviceName,
		transport:    transport,
		encoder:      encoder,
		handlerChans: make(map[string]map[*Binding]chan *Message),
	}
	transport.Handle(c.serviceName, c.handleMessage)
	return c
}

func (c *Client) Service(remoteServiceName string) *ServiceClient {
	return &ServiceClient{
		client:            c,
		remoteServiceName: remoteServiceName,
	}
}

func (c *Client) Bind(eventName string) *Binding {
	return newBinding(c, eventName)
}

func (c *Client) Close() error {
	return c.transport.Close()
}

func (c *Client) handleMessage(subject, sourceServiceName, replySubject string, reader io.Reader) {
	msg := &Message{
		sourceServiceName: sourceServiceName,
		replySubject:      replySubject,
		data:              reader,
		client:            c,
	}

	c.handlerChansMu.RLock()
	defer c.handlerChansMu.RUnlock()
	handlerChans, ok := c.handlerChans[subject]
	if ok {
		for _, ch := range handlerChans {
			ch <- msg
		}
	}
}
