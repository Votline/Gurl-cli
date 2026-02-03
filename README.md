# Gurl-cli

**Gurl-cli** is a high-performance, stateful HTTP & gRPC client for the command line.

Unlike standard tools (curl, postman), `gurl-cli` is designed for **request chaining**. It allows you to execute a sequence of requests where the output of one (e.g., Auth Token, Cookies) automatically feeds into the next, all defined in a single, readable file.

It runs on top of [Gurlf](https://github.com/Votline/Gurlf) — a custom zero-allocation configuration format that eliminates JSON escaping hell.

---

## 🚀 Key Features

* **Hybrid Transport:** First-class support for both **HTTP/1.1** and **gRPC** (via Reflection or Proto files).
* **Stateful Flows:** Automatically carries cookies between requests (like a browser jar).
* **Dynamic Chaining:** Inject responses from previous requests into headers or bodies using macro instructions (`{RESPONSE id=...}`).
* **Interactive Configs:** The tool writes responses **back into the configuration file**, making debugging and prototyping instant.
* **Zero-Allocation Parsing:** Custom parser optimized for low latency and minimal GC pressure.

---

## ⚡ Performance

Built for speed. The internal architecture utilizes **Ring Buffers** with spinlocks and aggressive object pooling to keep heap allocations near zero during hot paths.

**Parser Benchmarks (AMD Ryzen 7 5800U):**

| Operation | Time (ns/op) | Allocations |
| --- | --- | --- |
| **ParseStream** | **311.7 ns/op** | **0 allocs/op** |
| HandleInstructions | 40.29 ns/op | 0 allocs/op |
| ParseHeaders | 12.01 ns/op | 0 allocs/op |
| ParseCookies | 64.68 ns/op | 0 allocs/op |
| ParseBody | 18.55 ns/op | 0 allocs/op |

*> The parser handles complex instruction injection and multiline extraction without generating garbage.*

---

## 🛠 Installation

```bash
go install github.com/Votline/Gurl-cli@latest

```

---

## 📖 Configuration & Syntax

Configs are written in `.gurlf` format. It supports multiline strings (JSON/XML) natively without escaping.

### 1. HTTP Request with Embedded JSON

```bash
[http_config]
URL: http://localhost:8080/api/users
Method: POST
Headers: Content-Type: application/json
Body: `
    {
        "username": "admin",
        "role": "superuser"
    }
`
# The CLI will auto-fill the response here after execution
Response:
[\http_config]

```

### 2. Request Chaining (The Power Move)

Extract a token from Request #1 and use it in Request #2.

```bash
# Request 1: Login
[login_req]
URL: http://localhost:8080/auth/login
Method: POST
Body: `{ "user": "admin", "pass": "12345" }`
Type: http
# Response contains: {"token": "eyJh..."}
Response: `{"token": "eyJh..."}`
[\login_req]

# Request 2: Protected Resource
[get_data]
URL: http://localhost:8080/api/protected
Method: GET
Headers: `
    {
        Content-Type: application/json
        # Inject token from Request #1 (index 0)
        Authorization: Bearer {RESPONSE id=0 json:token}
    }
`
Type: http
[\get_data]

```

### 3. gRPC Support

Supports both `proto` file parsing and Server Reflection.

```bash
[grpc_config]
Target: localhost:50052
Endpoint: users.UserService/RegUser
# Point to your local proto file
ProtoPath: ../protos/user-service.proto
ImportPaths: /path/to/dependencies/
Data: `
    {
        "name": "John Doe",
        "email": "john@example.com"
    }
`
Type: grpc
[\grpc_config]

```

### 4. Smart Cookie Management

Control how cookies are shared between requests using the `{COOKIES}` instruction.

* **Default:** Cookies are stored in an in-memory jar and passed sequentially.
* **Explicit:** Import cookies from a specific request or file.

```bash
[auth_request]
# ... generates cookies ...
[\auth_request]

[next_step]
# Option A: Inherit from specific request ID
CookieIn: `{COOKIES id=0}`

# Option B: Load from external cookie jar file
CookieIn: `{COOKIES id=file}` 
[\next_step]

```

### 5. Config Templates (Repeat)

Don't rewrite huge payloads. Use `[repeat]` to inherit a config and patch specific fields.

```bash
[base_req]
ID: 0
URL: http://api.com/v1/data
Body: `{ "large": "payload" }`
[\base_req]

[repeat]
Target_ID: 0
# Patch just the name field in the body
Replace: `
    [patch]
    Body: `{ "large": "payload", "patched": true }`
    [\patch]
`
Type: repeat
[\repeat]

```

---

## 🏗 Architecture

For the curious engineers, `gurl-cli` is built on a custom concurrency model:

1. **Transport Layer:** Uses a custom `http.Transport` and `grpc.ClientConn` wrapped in an atomic worker pool.
2. **Memory Management:** Heavy use of pooling for byte buffers and context objects.
3. **Ring Buffers:** Internal communication between the Parser, Executor, and File Writer uses lock-free inspired Ring Buffers (`internal/buffer`) with atomic cursors to maximize throughput.
4. **In-Place IO:** The file writer uses a sophisticated seek/flush mechanism to update your config file with responses in real-time without corrupting the syntax.

---

## License

- **License:** This project is licensed under  [MIT](LICENSE)
- **Third-party Licenses:** The full license texts are available in the  [licenses/](licenses/)
