# go-rate-limiter

A production-grade HTTP rate limiter middleware for Go using the **Token Bucket** algorithm. Plug it into any `net/http` server in seconds.

## Features

- ✅ Token Bucket algorithm — smooth, burst-friendly rate limiting
- ✅ Per-IP tracking with automatic stale bucket cleanup
- ✅ Drop-in `http.Handler` middleware — works with any Go HTTP server
- ✅ Thread-safe with fine-grained mutex locking
- ✅ JSON error responses with `Retry-After` header
- ✅ Zero external dependencies

## Quick Start

```bash
git clone https://github.com/yourusername/go-rate-limiter
cd go-rate-limiter
go run main.go
```

Server starts at `http://localhost:8080`

## How It Works

Each unique IP gets its own token bucket:
- Bucket starts **full** (default: 10 tokens)
- Each request **consumes** 1 token
- Tokens **refill** at a configurable rate (default: 2/sec)
- When empty → `429 Too Many Requests`

```
Request → Check IP bucket → Has tokens? → Allow ✅
                                ↓ No
                           Reject 429 ❌
```

## Configuration

```go
// capacity: max burst size
// refillRate: tokens added per second
limiter := NewRateLimiter(10, 2)
```

| Parameter    | Default | Description                    |
|--------------|---------|--------------------------------|
| `capacity`   | 10      | Max requests in a burst        |
| `refillRate` | 2.0     | Tokens refilled per second     |

## API

| Endpoint  | Description          |
|-----------|----------------------|
| `GET /`   | Example rate-limited route |
| `GET /health` | Health check (also rate-limited) |

## Usage as Middleware

```go
limiter := NewRateLimiter(100, 10)

mux := http.NewServeMux()
mux.HandleFunc("/api/data", yourHandler)

http.ListenAndServe(":8080", limiter.Middleware(mux))
```

## Tech Stack

- **Language**: Go 1.21+
- **Dependencies**: None (stdlib only)

## License

MIT
