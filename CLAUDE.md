# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Redis clone implementation for the CodeCrafters challenge, written in Go. The project implements a subset of Redis commands including basic string operations, list operations, stream operations, and transactions.

## Development Commands

- **Build and run**: `./your_program.sh` or manually:
  ```bash
  go build -o /tmp/codecrafters-build-redis-go app/*.go
  /tmp/codecrafters-build-redis-go
  ```
- **Test**: `go test ./...`
- **Format**: `go fmt ./...`
- **Vet**: `go vet ./...`
- **Run locally**: The server listens on port 6379 by default

## Architecture

### Core Components

- **main.go**: Entry point with TCP server, connection handling, and transaction management
- **command.go**: Command handlers for Redis commands (PING, SET, GET, RPUSH, LPUSH, XADD, etc.)
- **resp.go**: RESP (Redis Serialization Protocol) parser and serializer
- **store.go**: Thread-safe in-memory data store with entity management
- **entity/**: Data type implementations (StringEntity, ListEntity, StreamEntity)

### Key Architecture Patterns

- **Event-driven**: Commands are processed through a channel-based event system
- **Transaction wrapper**: All commands except MULTI/EXEC/DISCARD are wrapped with transaction logic
- **Entity abstraction**: Different data types implement the Entity interface
- **Thread-safe store**: Uses RWMutex for concurrent access to the data store

### Data Structures

- **Store**: Central data storage using `map[string]entity.Entity` with RWMutex
- **ListEntity**: Backed by QuickList for efficient list operations
- **StreamEntity**: Implements Redis streams with time-based IDs
- **Transaction**: Queues commands during MULTI/EXEC blocks

### Connection Management

- Each client connection gets its own `ConnContext` with transaction state
- Goroutine per connection for handling requests
- Blocking operations (BLPOP, XREAD) use notification channels

## Important Implementation Details

- RESP protocol parsing handles all Redis data types (strings, arrays, bulk strings, etc.)
- Expiration is checked lazily during get operations
- Blocking commands use Go channels for notifications
- Stream IDs follow Redis format: `<milliseconds>-<sequence>`
- Transaction commands are queued and executed atomically on EXEC