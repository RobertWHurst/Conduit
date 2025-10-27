package conduit

import (
	"io"
	"sync"
)

// MaxDecodeSize is the maximum number of bytes that will be read when decoding a message.
// The default is 5MB. Set to -1 to disable the limit.
var MaxDecodeSize = int64(1024 * 1024 * 5)

// Conduit represents a service in the messaging system. Each instance has a unique name
// and can send/receive messages using a configured transport and encoder.
type Conduit struct {
	serviceName         string
	transport           Transport
	encoder             Encoder
	handlerChansMu      sync.RWMutex
	handlerChans        map[string]map[*Binding]chan *Message
	queueHandlerChansMu sync.RWMutex
	queueHandlerChans   map[string]map[*Binding]chan *Message
}

// New creates a new Conduit instance with the given service name, transport, and encoder.
// The service name identifies this service in the distributed system.
// The transport handles the underlying message delivery.
// The encoder serializes and deserializes message payloads.
func New(serviceName string, transport Transport, encoder Encoder) *Conduit {
	c := &Conduit{
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
func (c *Conduit) Service(remoteServiceName string) *ServiceClient {
	return &ServiceClient{
		conduit:           c,
		remoteServiceName: remoteServiceName,
	}
}

// Self creates a client for communicating with instances of the service
// represented by this Conduit.
func (c *Conduit) Self() *ServiceClient {
	return c.Service(c.serviceName)
}

// Bind creates a binding that subscribes to broadcast messages on the given subject.
// All instances of the service receive each message (fan-out).
// Use this for events that all instances should process.
// The binding must be closed when no longer needed to free resources.
func (c *Conduit) Bind(eventName string) *Binding {
	return newBinding(c, BindTypeNormal, eventName)
}

// BindOnce creates a binding that subscribes to broadcast messages on the given subject.
// All instances of the service receive each message (fan-out).
// The binding will automatically unbind after receiving one message.
// Use this for events that only need to be processed once.
// The binding must be closed when no longer needed to free resources.
func (c *Conduit) BindOnce(eventName string) *Binding {
	return newBinding(c, BindTypeOnce, eventName)
}

// QueueBind creates a binding that subscribes to load-balanced messages on the given subject.
// Only one instance of the service receives each message (round-robin).
// Use this for work that should be distributed across instances.
// The binding must be closed when no longer needed to free resources.
func (c *Conduit) QueueBind(eventName string) *Binding {
	return newBinding(c, BindTypeQueue, eventName)
}

// Close closes the conduit instance and cleans up resources.
func (c *Conduit) Close() error {
	return c.transport.Close()
}

func (c *Conduit) handleMessage(subject, sourceServiceName, replySubject string, reader io.Reader) {
	msg := &Message{
		sourceServiceName: sourceServiceName,
		replySubject:      replySubject,
		data:              reader,
		conduit:           c,
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

func (c *Conduit) handleQueueMessage(subject, sourceServiceName, replySubject string, reader io.Reader) {
	msg := &Message{
		sourceServiceName: sourceServiceName,
		replySubject:      replySubject,
		data:              reader,
		conduit:           c,
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
