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
  - [Multiple Services](#multiple-services)
  - [Custom Encoders](#custom-encoders)
  - [Custom Transports](#custom-transports)
  - [Error Handling](#error-handling)
- [Performance](#performance)
- [Architecture](#architecture)
- [Testing](#testing)
- [Help Welcome](#help-welcome)
- [License](#license)
- [Related Projects](#related-projects)

## Features

- 🚀 **High Performance** - Efficient message routing with streaming support for large payloads
- 🔌 **Pluggable Transports** - Built-in NATS support with interface for custom transports
- 📦 **Multiple Encoders** - JSON, MessagePack, and Protocol Buffers out of the box
- 🌊 **Streaming First** - `io.Reader`-based design for memory-efficient large message handling
- 🔄 **Request/Reply** - Built-in request/reply pattern with timeout support
- 📡 **Event Driven** - Bind handlers to events with automatic message routing
- 🎯 **Type Safe** - Work with strongly-typed Go structs or raw bytes
- 🛡️ **Backpressure** - Buffered channels provide natural backpressure for slow consumers
- 🧩 **Composable** - Simple interfaces make it easy to extend with custom encoders and transports

## Installation

```bash
go get github.com/RobertWHurst/conduit
```

For NATS transport:
```bash
go get github.com/nats-io/nats.go
```

## Quick Start

Here's a simple example showing two services communicating over NATS:

```go
package main

import (
    "fmt"
    "github.com/RobertWHurst/conduit"
    "github.com/RobertWHurst/conduit/encoders/json"
    "github.com/RobertWHurst/conduit/transports/nats"
    natsgo "github.com/nats-io/nats.go"
)

type GetUserRequest struct {
    UserID int `json:"user_id"`
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    // Connect to NATS
    nc, _ := natsgo.Connect("nats://localhost:4222")
    
    // Create user service
    userService := conduit.NewClient(
        "user-service",
        nats.NewNatsTransport(nc),
        json.New(),
    )
    
    // Bind handler for user requests
    userService.Bind("user.get").To(func(msg *conduit.Message) {
        var req GetUserRequest
        msg.Into(&req)
        
        user := User{ID: req.UserID, Name: "Alice"}
        msg.Reply(user)
    })
    
    // Create API service that calls user service
    apiService := conduit.NewClient(
        "api-service",
        nats.NewNatsTransport(nc),
        json.New(),
    )
    
    // Send request to user service
    var user User
    apiService.Service("user-service").Request("user.get", GetUserRequest{UserID: 123}).Into(&user)
    fmt.Printf("Got user: %s\n", user.Name)
}
```

## Core Concepts

### Client

The Client is the central object for sending and receiving messages. Each client represents a service and has a unique name that other services use to communicate with it.

Clients are created with three components: a service name, a transport (like NATS), and an encoder (like JSON). The service name identifies this client in the distributed system, the transport handles the underlying communication, and the encoder serializes/deserializes messages.

```go
client := conduit.NewClient(
    "my-service",
    nats.NewNatsTransport(natsConn),
    json.New(),
)
defer client.Close()
```

### Service Communication

Services communicate by creating service clients that target specific remote services. A service client is created by calling `client.Service()` with the target service name.

Service clients provide methods for sending messages (`Send`), making requests with replies (`Request`), and setting custom timeouts or contexts.

```go
// Create a client to communicate with "user-service"
userClient := client.Service("user-service")

// Fire-and-forget
userClient.Send("user.created", User{ID: 123, Name: "Alice"})

// Request with reply
var user User
userClient.Request("user.get", GetUserRequest{UserID: 123}).Into(&user)

// Request with custom timeout
userClient.RequestWithTimeout("user.get", req, 5*time.Second).Into(&user)

// Request with context
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
userClient.RequestWithCtx(ctx, "user.get", req).Into(&user)
```

### Message Encoding

Conduit uses encoder interfaces to serialize and deserialize messages. This means you can send and receive strongly-typed Go structs without manually marshaling JSON or protobuf.

Encoders handle three types of values automatically:

- **Structs** - Automatically marshaled using the encoder (JSON, MessagePack, Protocol Buffers)
- **Strings** - Sent as-is without encoding
- **Byte slices** - Sent as-is without encoding
- **io.Reader** - Streamed directly without buffering

This design makes it easy to send small structured messages or large binary streams using the same API.

```go
// Send a struct - automatically encoded
client.Service("user-service").Send("user.created", User{ID: 123, Name: "Alice"})

// Send a string
client.Service("log-service").Send("log.message", "User logged in")

// Send bytes
client.Service("data-service").Send("data.chunk", []byte{0x01, 0x02, 0x03})

// Stream a file
file, _ := os.Open("/path/to/large/file.dat")
client.Service("storage-service").Send("file.upload", file)
```

### Transports

Transports handle the underlying communication between services. Conduit ships with a NATS transport that provides reliable, high-performance messaging with streaming support.

Transports implement a simple interface, making it easy to add support for other message brokers like RabbitMQ, Redis, or Kafka.

The NATS transport uses a chunked streaming protocol that allows it to send messages of any size without loading them entirely into memory. Messages are split into 16KB chunks and streamed from sender to receiver.

```go
// Create NATS connection
nc, _ := natsgo.Connect("nats://localhost:4222")

// Use with client
client := conduit.NewClient("my-service", nats.NewNatsTransport(nc), json.New())
```

## Messaging Patterns

### Send (Fire and Forget)

Send is a one-way message that doesn't expect a reply. It's the fastest messaging pattern since it doesn't wait for acknowledgment beyond the transport-level confirmation.

Use Send for notifications, events, logging, and other cases where you don't need to know if the message was processed successfully.

```go
// Notify user service of login event
client.Service("user-service").Send("user.login", LoginEvent{
    UserID:    123,
    Timestamp: time.Now(),
    IPAddress: "192.168.1.1",
})

// Send log message
client.Service("log-service").Send("log.info", "User 123 logged in")
```

### Request/Reply

Request/Reply is a synchronous pattern where the sender waits for a response. The Request method blocks until a reply arrives or the timeout expires.

This is the pattern to use for queries, RPC calls, and any operation where you need a result from the remote service.

```go
// Make request and wait for reply
var user User
if err := client.Service("user-service").Request("user.get", GetUserRequest{UserID: 123}).Into(&user); err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %s\n", user.Name)
```

The default timeout is 30 seconds. Use `RequestWithTimeout` or `RequestWithCtx` for custom timeouts:

```go
// 5 second timeout
var user User
client.Service("user-service").RequestWithTimeout("user.get", req, 5*time.Second).Into(&user)

// Context-based cancellation
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
client.Service("user-service").RequestWithCtx(ctx, "user.get", req).Into(&user)
```

### Event Binding

Event binding lets services subscribe to messages on specific subjects. When a message arrives, Conduit routes it to all registered bindings for that subject.

Bindings provide two ways to handle messages:

- **Next()** - Blocking call that returns the next message. Use in a loop to process messages sequentially.
- **To()** - Non-blocking call that spawns a goroutine and calls your handler for each message.

```go
// Option 1: Process messages in a loop
binding := client.Bind("user.created")
go func() {
    for {
        var user User
        binding.Next().Into(&user)
        fmt.Printf("User created: %s\n", user.Name)
    }
}()

// Option 2: Use a handler function
client.Bind("user.created").To(func(msg *conduit.Message) {
    var user User
    msg.Into(&user)
    fmt.Printf("User created: %s\n", user.Name)
})
```

Close bindings when you're done to free resources:

```go
binding := client.Bind("user.created")
defer binding.Close()
```

### Queue Binding

Queue binding distributes messages across multiple service instances, ensuring each message is processed by only one instance. This is useful for load balancing work across multiple instances.

The API is identical to regular bindings, but uses `QueueBind()` instead of `Bind()`. The transport ensures only one instance receives each message through queue group semantics.

```go
// Multiple instances of the service can bind to the same queue
client.QueueBind("work.process").To(func(msg *conduit.Message) {
    var work WorkItem
    msg.Into(&work)
    processWork(work)
})
```

**When to use each:**

- **Bind()** - Use for events that all instances should process (e.g., cache invalidation, configuration updates)
- **QueueBind()** - Use for work that should be distributed (e.g., job processing, request handling)

**Note:** Both `Bind()` and `QueueBind()` can be used on the same subject simultaneously. Bind() subscribers all receive every message, while QueueBind() subscribers share messages (only one receives each).

## Message Handling

### Decoding Messages

The `Into()` method decodes a message into a Go struct using the configured encoder. It automatically handles the message data and respects the `MaxDecodeSize` limit (default 5MB) to prevent memory exhaustion.

```go
client.Bind("user.get").To(func(msg *conduit.Message) {
    var req GetUserRequest
    if err := msg.Into(&req); err != nil {
        log.Printf("Failed to decode: %v", err)
        return
    }
    
    // Process request
    user := getUserByID(req.UserID)
    msg.Reply(user)
})
```

Change the decode size limit globally or per-client:

```go
// Global limit (affects all clients)
conduit.MaxDecodeSize = 10 * 1024 * 1024 // 10MB

// Per-client limit
client := conduit.NewClient("my-service", transport, encoder)
// MaxDecodeSize is used when calling msg.Into()
```

### Reading Raw Data

For large messages or streaming data, use the message as an `io.Reader` to avoid loading everything into memory at once.

The message implements `io.Reader`, so you can use it with standard library functions like `io.Copy`, `io.ReadAll`, or read incrementally in a loop.

```go
client.Bind("file.upload").To(func(msg *conduit.Message) {
    file, _ := os.Create("/tmp/upload")
    defer file.Close()
    
    // Stream message data directly to file
    io.Copy(file, msg)
    
    log.Println("File uploaded")
})
```

### Replying to Messages

Messages that arrive via bindings can be replied to using the `Reply()` method. This sends a message back to the original sender on the reply subject embedded in the request.

Only messages that were sent via `Request()` have reply subjects. Messages sent via `Send()` cannot be replied to.

```go
client.Bind("user.get").To(func(msg *conduit.Message) {
    var req GetUserRequest
    msg.Into(&req)
    
    user := getUserByID(req.UserID)
    
    // Send reply back to requester
    if err := msg.Reply(user); err != nil {
        log.Printf("Failed to reply: %v", err)
    }
})
```

Like `Send()`, `Reply()` accepts structs, strings, byte slices, and `io.Reader` values.

## Built-in Encoders

### JSON Encoder

The JSON encoder uses Go's standard `encoding/json` package. It's human-readable, widely supported, and ideal for development, debugging, and APIs where compatibility is more important than performance.

```go
import "github.com/RobertWHurst/conduit/encoders/json"

client := conduit.NewClient(
    "my-service",
    transport,
    json.New(),
)
```

JSON is the best choice when:
- Messages need to be human-readable for debugging
- You're building public APIs or need broad client support
- Message size and performance are not critical concerns

### MessagePack Encoder

MessagePack is a binary encoding format that's significantly faster and more compact than JSON. It's ideal for high-throughput systems where performance matters.

**Why use MessagePack:**
- **5x faster** than JSON for serialization/deserialization
- **Smaller message sizes** - reduces bandwidth and memory usage
- **Binary efficiency** - better for high-performance systems
- **Drop-in replacement** - same API as JSON

```go
import "github.com/RobertWHurst/conduit/encoders/msgpack"

client := conduit.NewClient(
    "my-service",
    transport,
    msgpack.New(),
)
```

MessagePack is the best choice when:
- You need maximum throughput for high-volume messaging
- Bandwidth or memory is constrained
- All services are under your control (not a public API)

### Protocol Buffers Encoder

Protocol Buffers (protobuf) provides strong typing, schema validation, and excellent performance with compact binary encoding. It's the gold standard for cross-language communication in distributed systems.

**Why use Protocol Buffers:**
- **Type safety** - Strongly-typed schemas with compile-time validation
- **Language-agnostic** - Same `.proto` files work across Go, Java, Python, C++, and more
- **Schema evolution** - Add or deprecate fields without breaking existing services
- **Backward/forward compatibility** - Older clients work with newer servers
- **Industry standard** - Used by Google, Netflix, and thousands of companies

**Define your `.proto` schema:**

```protobuf
// user.proto
syntax = "proto3";
package myapp;
option go_package = "github.com/myuser/myapp/proto";

message GetUserRequest {
  int64 user_id = 1;
}

message User {
  int64 id = 1;
  string name = 2;
  string email = 3;
}
```

**Generate Go code:**

```bash
protoc --go_out=. --go_opt=paths=source_relative user.proto
```

**Use with Conduit:**

```go
import (
    "github.com/RobertWHurst/conduit/encoders/protobuf"
    pb "github.com/myuser/myapp/proto"
)

client := conduit.NewClient(
    "my-service",
    transport,
    protobuf.New(),
)

// Send protobuf messages
var user pb.User
client.Service("user-service").Request("user.get", &pb.GetUserRequest{UserId: 123}).Into(&user)
fmt.Printf("User: %s\n", user.Name)
```

Protocol Buffers is the best choice when:
- Building microservices architectures
- Supporting multiple programming languages
- You need type-safe APIs with schema validation
- Backward/forward compatibility is important

## Built-in Transports

### NATS Transport

The NATS transport provides reliable, high-performance messaging over NATS. It supports the full Conduit API including streaming, request/reply, and event patterns.

**Setup:**

```go
import (
    "github.com/RobertWHurst/conduit/transports/nats"
    natsgo "github.com/nats-io/nats.go"
)

// Connect to NATS server
nc, err := natsgo.Connect("nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}

// Create transport
transport := nats.NewNatsTransport(nc)

// Use with client
client := conduit.NewClient("my-service", transport, json.New())
```

**NATS Features:**
- **At-most-once delivery** - Fast, fire-and-forget messaging
- **Request/Reply** - Built-in request/reply pattern
- **Subject-based routing** - Messages are routed based on hierarchical subjects
- **Clustering** - High availability with NATS clusters
- **Security** - TLS encryption and authentication support

### Chunked Streaming

The NATS transport implements a chunked streaming protocol that allows messages of any size to be sent without loading them entirely into memory.

When you send an `io.Reader`, the transport:
1. Reads data in 16KB chunks
2. Sends each chunk with an index over NATS
3. Marks the final chunk with an EOF flag
4. The receiver assembles chunks into an `io.Reader` pipe

This design allows you to stream gigabyte-sized files over NATS without memory pressure, while still using the same simple API as small messages.

```go
// Stream large file without loading into memory
file, _ := os.Open("/path/to/large/file.dat")
defer file.Close()

client.Service("storage-service").Send("file.store", file)
```

The receiver can stream the data directly to disk or another destination:

```go
client.Bind("file.store").To(func(msg *conduit.Message) {
    outFile, _ := os.Create("/storage/uploaded-file.dat")
    defer outFile.Close()
    
    // Stream directly to disk - no buffering
    io.Copy(outFile, msg)
})
```

**Protocol Details:**
- Chunk size: 16KB (configurable via `nats.ChunkSize`)
- Timeout: 5 seconds for send acknowledgment (configurable via `nats.SendTimeout`)
- Subject format: `conduit.<service-name>`
- Namespacing: Automatic conversion of service names to valid NATS subjects

## Advanced Usage

### Multiple Services

A single application can run multiple Conduit clients, each representing a different service. This is useful for microservice architectures where one process handles multiple concerns.

Each client has its own identity and bindings, allowing you to organize your code around service boundaries while deploying as a single process.

```go
// Create multiple services in one process
userService := conduit.NewClient("user-service", transport, json.New())
authService := conduit.NewClient("auth-service", transport, json.New())
logService := conduit.NewClient("log-service", transport, json.New())

// Each service has independent bindings
userService.Bind("user.get").To(handleGetUser)
authService.Bind("auth.login").To(handleLogin)
logService.Bind("log.write").To(handleLog)

// Services can communicate with each other
var user User
authService.Service("user-service").Request("user.get", GetUserRequest{UserID: 123}).Into(&user)
```

### Custom Encoders

Implement the `Encoder` interface to add support for other serialization formats like CBOR, Avro, or custom binary protocols.

```go
type Encoder interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

Example custom encoder:

```go
type MyEncoder struct{}

func (e *MyEncoder) Encode(v any) ([]byte, error) {
    // Your encoding logic
    return encoded, nil
}

func (e *MyEncoder) Decode(data []byte, v any) error {
    // Your decoding logic
    return nil
}

// Use custom encoder
client := conduit.NewClient("my-service", transport, &MyEncoder{})
```

### Custom Transports

Implement the `Transport` interface to add support for other message brokers like RabbitMQ, Redis Pub/Sub, or Kafka.

```go
type Transport interface {
    Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error
    Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader)) error
    Close() error
}
```

The transport is responsible for:
- **Send** - Delivering messages to a specific service and subject, with optional reply subject
- **Handle** - Registering a handler that's called when messages arrive for this service
- **Close** - Cleaning up resources when the client shuts down

Example structure for a custom transport:

```go
type MyTransport struct {
    // Your transport state
}

func (t *MyTransport) Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
    // Read from reader and send to message broker
    data, _ := io.ReadAll(reader)
    // Send to your message broker...
    return nil
}

func (t *MyTransport) Handle(serviceName string, handler func(subject, sourceServiceName, replySubject string, reader io.Reader)) error {
    // Subscribe to messages for this service
    // When messages arrive, call handler with message data as io.Reader
    return nil
}

func (t *MyTransport) Close() error {
    // Clean up connections
    return nil
}
```

### Error Handling

Conduit returns errors from operations that can fail. Always check errors from `Into()`, `Reply()`, `Send()`, and `Request()` operations.

```go
// Check decode errors
var user User
if err := msg.Into(&user); err != nil {
    log.Printf("Failed to decode user: %v", err)
    return
}

// Check reply errors
if err := msg.Reply(response); err != nil {
    log.Printf("Failed to send reply: %v", err)
}

// Check send errors
if err := client.Service("user-service").Send("user.created", user); err != nil {
    log.Printf("Failed to send message: %v", err)
}

// Check request errors (timeout, decode failure)
var user User
if err := client.Service("user-service").RequestWithTimeout("user.get", req, 5*time.Second).Into(&user); err != nil {
    if err == context.DeadlineExceeded {
        log.Printf("Request timed out")
    } else {
        log.Printf("Request failed: %v", err)
    }
    return
}
```

Request errors are returned via `msg.Into()` and include:
- `context.DeadlineExceeded` - Request timed out
- `context.Canceled` - Request was canceled
- Decode errors - Failed to unmarshal response
- Transport errors - Network or broker failures

## Performance

Conduit is designed for high-performance messaging with minimal overhead:

- **Zero-copy streaming** - `io.Reader`-based design avoids unnecessary buffering
- **Efficient encoding** - MessagePack and Protocol Buffers provide fast serialization
- **Chunked transfers** - Large messages are streamed without loading into memory
- **Buffered channels** - 100-message buffer per binding provides natural backpressure
- **Connection reuse** - Transports maintain persistent connections to the broker

**Benchmarks** (coming soon)

## Architecture

Conduit has a clean, layered architecture:

```
┌─────────────────────────────────────────┐
│            Application Code             │
├─────────────────────────────────────────┤
│              Conduit Client             │
│  (Service naming, message routing)      │
├─────────────────────────────────────────┤
│              Encoder Layer              │
│     (JSON, MessagePack, Protobuf)       │
├─────────────────────────────────────────┤
│            Transport Layer              │
│    (NATS, RabbitMQ, Redis, Kafka)       │
└─────────────────────────────────────────┘
```

**Key Design Decisions:**

- **Streaming-first** - `io.Reader` throughout the stack for memory efficiency
- **Pluggable components** - Simple interfaces for encoders and transports
- **Minimal dependencies** - Core package uses only Go standard library
- **Backpressure** - Buffered channels prevent fast producers from overwhelming slow consumers

## Testing

(Coming soon)

## Help Welcome

Conduit is in active development and contributions are welcome! Areas where help would be appreciated:

- Additional transports (RabbitMQ, Redis, Kafka)
- Additional encoders (CBOR, Avro)
- Documentation improvements
- Performance benchmarks
- Example applications
- Test coverage

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [Navaros](https://github.com/RobertWHurst/Navaros) - HTTP router for Go with powerful pattern matching
- [Velaros](https://github.com/RobertWHurst/Velaros) - WebSocket framework with message routing
