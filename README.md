# Conduit

A lightweight, transport-agnostic messaging framework for Go. Build distributed systems with multiple encoding formats, streaming support, and pluggable transports like NATS.

[![Go Reference](https://pkg.go.dev/badge/github.com/RobertWHurst/conduit.svg)](https://pkg.go.dev/github.com/RobertWHurst/conduit)
[![Go Report Card](https://goreportcard.com/badge/github.com/RobertWHurst/conduit)](https://goreportcard.com/report/github.com/RobertWHurst/conduit)
[![License](https://img.shields.io/github/license/RobertWHurst/conduit)](LICENSE)
[![Sponsor](https://img.shields.io/static/v1?label=Sponsor&message=%E2%9D%A4&logo=GitHub&color=%23fe8e86)](https://github.com/sponsors/RobertWHurst)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
  - [Client](#client)
  - [Service Communication](#service-communication)
  - [Message Encoding](#message-encoding)
  - [Transports](#transports)
- [Messaging Patterns](#messaging-patterns)
  - [Send (Fire and Forget)](#send-fire-and-forget)
  - [Request/Reply](#requestreply)
  - [Event Binding](#event-binding)
  - [Queue Binding](#queue-binding)
- [Message Handling](#message-handling)
  - [Decoding Messages](#decoding-messages)
  - [Reading Raw Data](#reading-raw-data)
  - [Replying to Messages](#replying-to-messages)
- [Built-in Encoders](#built-in-encoders)
  - [JSON Encoder](#json-encoder)
  - [MessagePack Encoder](#messagepack-encoder)
  - [Protocol Buffers Encoder](#protocol-buffers-encoder)
- [Built-in Transports](#built-in-transports)
  - [NATS Transport](#nats-transport)
  - [Chunked Streaming](#chunked-streaming)
- [Advanced Usage](#advanced-usage)
  - [Custom Encoders](#custom-encoders)
  - [Custom Transports](#custom-transports)
- [Help Welcome](#help-welcome)
- [License](#license)
- [Related Projects](#related-projects)

## Features

- 📡 **Event-Driven Messaging** - Broadcast events across services with automatic routing
- 🌊 **Streaming Support** - Send messages of any size without loading into memory
- 🔌 **Pluggable Transports** - Built-in NATS support, easily add RabbitMQ, Kafka, or Redis
- 📦 **Multiple Encoders** - JSON, MessagePack, and Protocol Buffers included
- ⚖️ **Load Balancing** - Queue bindings distribute work across service instances
- 🔄 **Request/Reply** - Synchronous request/reply pattern when you need it

## Installation

```bash
go get github.com/RobertWHurst/conduit
```

For NATS transport:
```bash
go get github.com/nats-io/nats.go
```

## Quick Start

Here's a simple example showing event-driven communication between services over NATS:

**User Service** (broadcasts events):

```go
// Connect to NATS
nc, _ := natsgo.Connect("nats://localhost:4222")

// Create client
client := conduit.NewClient("user-service", nats.NewNatsTransport(nc), json.New())

// Broadcast user.created event when a user signs up
client.Service("notification-service").Send("user.created", UserCreatedEvent{
    UserID: 123,
    Email:  "alice@example.com",
    Name:   "Alice",
})
```

**Notification Service** (listens for events):

```go
// Connect to NATS
nc, _ := natsgo.Connect("nats://localhost:4222")

// Create client
client := conduit.NewClient("notification-service", nats.NewNatsTransport(nc), json.New())

// Listen for user.created events
client.Bind("user.created").To(func(msg *conduit.Message) {
    var event UserCreatedEvent
    msg.Into(&event)
    
    // Send welcome email
    sendWelcomeEmail(event.Email, event.Name)
})
```

## Core Concepts

### Client

The client allows sending and receiving data from different services. A client takes the name of the service it represents, a transport for facilitating communication, and an encoder for encoding and decoding structs.

Each service should have one client instance. The service name identifies your service to others in the distributed system. The transport handles the underlying message delivery (like NATS). The encoder serializes and deserializes your data structures.

```go
client := conduit.NewClient(
    "my-service",
    nats.NewNatsTransport(natsConn),
    json.New(),
)
defer client.Close()
```

### Service Communication

To send messages to another service, create a service client using `client.Service()` with the target service name. This returns a `ServiceClient` that provides methods for sending one-way messages or making request/reply calls.

```go
// Create a service client for "notification-service"
notificationService := client.Service("notification-service")

// Send one-way message
notificationService.Send("email.send", EmailRequest{
    To:      "user@example.com",
    Subject: "Welcome!",
})

// Request with reply (blocks until response or timeout)
var result EmailResult
notificationService.Request("email.send", emailReq).Into(&result)

// Custom timeout
notificationService.RequestWithTimeout("email.send", emailReq, 5*time.Second).Into(&result)

// Context-based cancellation
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
notificationService.RequestWithCtx(ctx, "email.send", emailReq).Into(&result)
```

### Message Encoding

Encoders serialize and deserialize messages automatically. Send strongly-typed Go structs without manual marshaling.

Encoders handle five types of values:

- **Structs** - Marshaled using the encoder (JSON, MessagePack, Protocol Buffers)
- **Strings** - Sent as-is without encoding
- **Byte slices** - Sent as-is without encoding
- **io.Reader** - Streamed directly without buffering
- **nil** - Send events without payload data

```go
// Send a struct
client.Service("notification-service").Send("user.created", User{ID: 123, Name: "Alice"})

// Send a string
client.Service("log-service").Send("log.info", "User logged in")

// Send bytes
client.Service("analytics-service").Send("event.track", []byte{0x01, 0x02, 0x03})

// Stream a file
file, _ := os.Open("report.pdf")
client.Service("storage-service").Send("file.upload", file)

// Send event without payload
client.Service("cache-service").Send("cache.invalidate", nil)
```

### Transports

Transports handle the underlying message delivery between services. Conduit includes a NATS transport with support for reliable messaging and streaming.

The transport interface is simple, making it straightforward to add support for other brokers like RabbitMQ, Redis, or Kafka.

The NATS transport uses a chunked streaming protocol to send messages of any size without loading them into memory. Messages are split into 16KB chunks and streamed between services.

```go
nc, _ := natsgo.Connect("nats://localhost:4222")
client := conduit.NewClient("my-service", nats.NewNatsTransport(nc), json.New())
```

## Messaging Patterns

### Send (Fire and Forget)

Send delivers one-way messages without waiting for a reply. This is ideal for broadcasting events, logging, and notifications where you don't need confirmation of processing.

```go
// Broadcast login event
client.Service("analytics-service").Send("user.login", LoginEvent{
    UserID:    123,
    Timestamp: time.Now(),
    IPAddress: "192.168.1.1",
})

// Send log message
client.Service("log-service").Send("log.info", "User 123 logged in")
```

### Request/Reply

Request/Reply is a synchronous pattern where the sender waits for a response. Use this when you need a result back from another service.

```go
// Make request and wait for reply (30 second default timeout)
var result ProcessResult
if err := client.Service("worker-service").Request("job.process", job).Into(&result); err != nil {
    log.Fatal(err)
}

// Custom timeout
client.Service("worker-service").RequestWithTimeout("job.process", job, 5*time.Second).Into(&result)

// Context-based cancellation
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
client.Service("worker-service").RequestWithCtx(ctx, "job.process", job).Into(&result)
```

### Event Binding

Bind to subjects to receive messages. All service instances with the same binding will receive every message (broadcast).

Use `Next()` to process messages in a loop, or `To()` to handle messages with a callback:

```go
// Option 1: Process messages in a loop
binding := client.Bind("user.created")
go func() {
    for {
        msg := binding.Next()
        var event UserCreatedEvent
        msg.Into(&event)
        updateCache(event)
    }
}()

// Option 2: Use a handler function
client.Bind("user.created").To(func(msg *conduit.Message) {
    var event UserCreatedEvent
    msg.Into(&event)
    updateCache(event)
})
```

Close bindings when done:

```go
binding := client.Bind("user.created")
defer binding.Close()
```

### Queue Binding

Queue bindings distribute messages across service instances - only one instance receives each message. Use this for load balancing work.

```go
// Each message goes to only one instance
client.QueueBind("job.process").To(func(msg *conduit.Message) {
    var job Job
    msg.Into(&job)
    processJob(job)
})
```

**When to use:**

- **Bind()** - All instances receive the message (cache invalidation, config updates)
- **QueueBind()** - One instance receives the message (job processing, work distribution)

Both can be used on the same subject simultaneously.

## Message Handling

### Decoding Messages

Use `Into()` to decode messages into Go structs. The decoder respects `MaxDecodeSize` (default 5MB) to prevent memory exhaustion.

```go
client.Bind("order.created").To(func(msg *conduit.Message) {
    var event OrderCreatedEvent
    if err := msg.Into(&event); err != nil {
        log.Printf("Failed to decode: %v", err)
        return
    }
    
    processOrder(event)
})
```

Change the decode size limit:

```go
conduit.MaxDecodeSize = 10 * 1024 * 1024 // 10MB
```

### Reading Raw Data

Messages implement `io.Reader` for streaming large data without loading it into memory.

```go
client.Bind("file.upload").To(func(msg *conduit.Message) {
    file, _ := os.Create("/tmp/upload")
    defer file.Close()
    
    io.Copy(file, msg)
    log.Println("File uploaded")
})
```

### Replying to Messages

Use `Reply()` to respond to requests. Only messages sent via `Request()` have reply subjects - messages from `Send()` cannot be replied to.

```go
client.Bind("job.process").To(func(msg *conduit.Message) {
    var job Job
    msg.Into(&job)
    
    result := processJob(job)
    
    if err := msg.Reply(result); err != nil {
        log.Printf("Failed to reply: %v", err)
    }
})
```

`Reply()` accepts structs, strings, byte slices, and `io.Reader` values.

## Built-in Encoders

### JSON Encoder

JSON encoding is human-readable and widely supported. Good for development and debugging.

```go
import "github.com/RobertWHurst/conduit/encoders/json"

client := conduit.NewClient("my-service", transport, json.New())
```

Use JSON when:
- Human-readable messages are important
- Broad compatibility is needed
- Performance is not critical

### MessagePack Encoder

MessagePack is a fast, compact binary format - approximately 5x faster than JSON with smaller message sizes.

```go
import "github.com/RobertWHurst/conduit/encoders/msgpack"

client := conduit.NewClient("my-service", transport, msgpack.New())
```

Use MessagePack when:
- High throughput is needed
- Bandwidth or memory is constrained
- All services are under your control

### Protocol Buffers Encoder

Protocol Buffers provides type safety, schema validation, and cross-language compatibility.

**Define a schema:**

```protobuf
syntax = "proto3";

message UserCreatedEvent {
  int64 user_id = 1;
  string email = 2;
  string name = 3;
}
```

**Generate Go code:**

```bash
protoc --go_out=. --go_opt=paths=source_relative events.proto
```

**Use with Conduit:**

```go
import (
    "github.com/RobertWHurst/conduit/encoders/protobuf"
    pb "github.com/myuser/myapp/proto"
)

client := conduit.NewClient("my-service", transport, protobuf.New())

// Send protobuf message
client.Service("notification-service").Send("user.created", &pb.UserCreatedEvent{
    UserId: 123,
    Email:  "alice@example.com",
    Name:   "Alice",
})
```

Use Protocol Buffers when:
- Type-safe schemas are needed
- Supporting multiple languages
- Backward/forward compatibility is important

## Built-in Transports

### NATS Transport

The NATS transport provides reliable, high-performance messaging with support for streaming, request/reply, and events.

```go
import (
    "github.com/RobertWHurst/conduit/transports/nats"
    natsgo "github.com/nats-io/nats.go"
)

nc, _ := natsgo.Connect("nats://localhost:4222")
transport := nats.NewNatsTransport(nc)
client := conduit.NewClient("my-service", transport, json.New())
```

Features:
- At-most-once delivery
- Subject-based routing
- Clustering for high availability
- TLS encryption and authentication

### Chunked Streaming

The NATS transport streams messages of any size without loading them into memory. Data is sent in 16KB chunks.

```go
// Stream large file
file, _ := os.Open("large-file.dat")
client.Service("storage-service").Send("file.store", file)
```

Receiver streams directly to disk:

```go
client.Bind("file.store").To(func(msg *conduit.Message) {
    outFile, _ := os.Create("uploaded-file.dat")
    defer outFile.Close()
    io.Copy(outFile, msg)
})
```

Protocol details:
- Chunk size: 16KB (configurable via `nats.ChunkSize`)
- Send timeout: 5 seconds (configurable via `nats.SendTimeout`)
- Subject format: `conduit.<service-name>`

## Advanced Usage

### Custom Encoders

Implement the `Encoder` interface to add support for other formats:

```go
type Encoder interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}

type MyEncoder struct{}

func (e *MyEncoder) Encode(v any) ([]byte, error) {
    // Your encoding logic
    return encoded, nil
}

func (e *MyEncoder) Decode(data []byte, v any) error {
    // Your decoding logic
    return nil
}

client := conduit.NewClient("my-service", transport, &MyEncoder{})
```

### Custom Transports

Implement the `Transport` interface to add support for other message brokers:

```go
type Transport interface {
    Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error
    Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))
    HandleQueue(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))
    Close() error
}

type MyTransport struct{}

func (t *MyTransport) Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
    data, _ := io.ReadAll(reader)
    // Send to your message broker
    return nil
}

func (t *MyTransport) Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
    // Subscribe to broadcast messages
}

func (t *MyTransport) HandleQueue(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader)) {
    // Subscribe to queue messages
}

func (t *MyTransport) Close() error {
    return nil
}
```

## Help Welcome

If you want to support this project with coffee money, it's greatly appreciated.

[![sponsor](https://img.shields.io/static/v1?label=Sponsor&message=%E2%9D%A4&logo=GitHub&color=%23fe8e86)](https://github.com/sponsors/RobertWHurst)

If you're interested in providing feedback or would like to contribute, please feel free to do so. I recommend first [opening an issue][feature-request] expressing your feedback or intent to contribute a change. From there we can consider your feedback or guide your contribution efforts. Any and all help is greatly appreciated.

Thank you!

[feature-request]: https://github.com/RobertWHurst/Conduit/issues/new?template=feature_request.md

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [Navaros](https://github.com/RobertWHurst/Navaros) - HTTP framework for Go with powerful pattern matching
- [Velaros](https://github.com/RobertWHurst/Velaros) - WebSocket framework for Go with message routing
- [Zephyr](https://github.com/TelemetryTV/Zephyr) - Microservice framework built on Navaros with service discovery and streaming
- Eurus - WebSocket API gateway framework (upcoming, integrates with Velaros)
