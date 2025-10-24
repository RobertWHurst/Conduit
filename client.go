package conduit

import (
	"io"
	"sync"
)

// MaxDecodeSize is the maximum number of bytes that will be read when decoding a message.
// The default is 5MB. Set to -1 to disable the limit.
var MaxDecodeSize = int64(1024 * 1024 * 5)

// Client represents a service in the messaging system. Each client has a unique name
// and can send/receive messages using a configured transport and encoder.
type Client struct {
	serviceName         string
	transport           Transport
	encoder             Encoder
	handlerChansMu      sync.RWMutex
	handlerChans        map[string]map[*Binding]chan *Message
	queueHandlerChansMu sync.RWMutex
	queueHandlerChans   map[string]map[*Binding]chan *Message
}

// NewClient creates a new client with the given service name, transport, and encoder.
// The service name identifies this client in the distributed system.
// The transport handles the underlying message delivery.
// The encoder serializes and deserializes message payloads.
func NewClient(serviceName string, transport Transport, encoder Encoder) *Client {
	c := &Client{
		serviceName:       serviceName,
		transport:         transport,
		encoder:           encoder,
		handlerChans:      make(map[string]map[*Binding]chan *Message),
		queueHandlerChans: make(map[string]map[*Binding]chan *Message),
	}
	transport.Handle(c.serviceName, c.handleMessage)
	transport.HandleQueue(c.serviceName, c.handleQueueMessage)
	return c
}

// Service creates a client for communicating with a remote service.
// The returned ServiceClient provides methods for sending messages and making requests.
func (c *Client) Service(remoteServiceName string) *ServiceClient {
	return &ServiceClient{
		client:            c,
		remoteServiceName: remoteServiceName,
	}
}

// Bind creates a binding that subscribes to broadcast messages on the given subject.
// All instances of the service receive each message (fan-out).
// Use this for events that all instances should process.
// The binding must be closed when no longer needed to free resources.
func (c *Client) Bind(eventName string) *Binding {
	return newBinding(c, BindTypeNormal, eventName)
}

// BindOnce creates a binding that subscribes to broadcast messages on the given subject.
// All instances of the service receive each message (fan-out).
// The binding will automatically unbind after receiving one message.
// Use this for events that only need to be processed once.
// The binding must be closed when no longer needed to free resources.
func (c *Client) BindOnce(eventName string) *Binding {
	return newBinding(c, BindTypeOnce, eventName)
}

// QueueBind creates a binding that subscribes to load-balanced messages on the given subject.
// Only one instance of the service receives each message (round-robin).
// Use this for work that should be distributed across instances.
// The binding must be closed when no longer needed to free resources.
func (c *Client) QueueBind(eventName string) *Binding {
	return newBinding(c, BindTypeQueue, eventName)
}

// Close closes the client and cleans up resources.
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

func (c *Client) handleQueueMessage(subject, sourceServiceName, replySubject string, reader io.Reader) {
	msg := &Message{
		sourceServiceName: sourceServiceName,
		replySubject:      replySubject,
		data:              reader,
		client:            c,
	}

	c.queueHandlerChansMu.RLock()
	defer c.queueHandlerChansMu.RUnlock()
	handlerChans, ok := c.queueHandlerChans[subject]
	if ok {
		for _, ch := range handlerChans {
			ch <- msg
		}
	}
}
