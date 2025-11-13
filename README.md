# Gurl-cli 🚀

[![Go Version](https://img.shields.io/badge/Go-1.24.5-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](https://opensource.org/licenses/MIT)
[![HTTP Ready](https://img.shields.io/badge/HTTP-Ready-%23007EC6?style=flat-square&logo=internetexplorer)](https://developer.mozilla.org/en-US/docs/Web/HTTP)
[![gRPC Ready](https://img.shields.io/badge/gRPC-Ready-%23007EC6?style=flat-square&logo=google)](https://grpc.io/)
[![Protobuf Support](https://img.shields.io/badge/Protobuf-Supported-green?style=flat-square&logo=protobuf)](https://protobuf.dev/)
[![JSON Configs](https://img.shields.io/badge/JSON-Configs-yellow?style=flat-square&logo=json)](https://www.json.org/)
[![Response Chaining](https://img.shields.io/badge/Response-Chaining-purple?style=flat-square)](https://github.com/Votlines/Gurl-cli)
[![Config Reuse](https://img.shields.io/badge/Config-Reuse-orange?style=flat-square)](https://github.com/Votline/Gulr-cli)

**Supercharged curl/grpcurl with config chaining and response templating.**  
Stop memorizing complex flags — save them as reusable JSON configs and hit run.

---



## Why this exists

**Typing long** curl/grpcurl incantations **sucks**. Gurl turns them into **tiny**, **reusable** configs you can **chain**:
- HTTP → HTTP, gRPC → gRPC, or mix them.
- Responses from earlier steps can feed later steps via simple placeholders.
- **No curl/grpcurl required** — native Go implementation using:
  - `net/http` for HTTP requests  
  - `google.golang.org/grpc` for gRPC calls
  - `github.com/jhump/protoreflect` for protobuf introspection

## AND

- **🔁 Response Placeholders Everywhere** - Use `{RESPONSE}` in URLs, headers, and bodies
- **🔄 Repeated Configs** - Duplicate and modify existing configs with `replace` fields  
- **♻️ Loop Processing** - All placeholders in a request are now properly processed
- **🆔 Automatic ID Management** - IDs are now enforced to prevent conflicts

---

## Quick Start

```bash
# 1) Generate a starter config
gurl-cli --config-create --config=my_config

# 2) Edit it with your endpoints
vim my_config.json

# 3) Run it
go run main.go --config=my_config.json
```

### Example: Chaining Requests

```json
[
  {
    "id": "1",
    "type": "http",
    "url": "http://localhost:8080/api/auth",
    "method": "POST",
    "body": {"user": "test", "pass": "test"},
    "response": "-"
  },
  {
    "id": "2", 
    "type": "http",
    "url": "http://localhost:8080/api/data/{RESPONSE id=1 json:user_id}",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {RESPONSE id=1 json:token}"
    },
    "response": "-"
  }
]
```

### Example: Repeated Configs

```json
{
  "type": "repeated",
  "repeated_id": "1",
  "replace": {
    "user": "different_user"
  }
}
```

---

## 📚 Documentation

- **Full Guide:** See [`GUIDE.md`](GUIDE.md) for detailed examples and advanced features
- **License:** This project is licensed under  [MIT](LICENSE)
- **Third-party Licenses:** The full license texts are available in the  [licenses/](licenses/)
