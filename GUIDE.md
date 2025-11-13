# Gurl-cli GUIDE 📖

[![Go Version](https://img.shields.io/badge/Go-1.24.5-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](https://opensource.org/licenses/MIT)
[![HTTP Ready](https://img.shields.io/badge/HTTP-Ready-%23007EC6?style=flat-square&logo=internetexplorer)](https://developer.mozilla.org/en-US/docs/Web/HTTP)
[![gRPC Ready](https://img.shields.io/badge/gRPC-Ready-%23007EC6?style=flat-square&logo=google)](https://grpc.io/)
[![Protobuf Support](https://img.shields.io/badge/Protobuf-Supported-green?style=flat-square&logo=protobuf)](https://protobuf.dev/)
[![JSON Configs](https://img.shields.io/badge/JSON-Configs-yellow?style=flat-square&logo=json)](https://www.json.org/)
[![Response Chaining](https://img.shields.io/badge/Response-Chaining-purple?style=flat-square)](https://github.com/Votlines/Gurl-cli)
[![Config Reuse](https://img.shields.io/badge/Config-Reuse-orange?style=flat-square)](https://github.com/Votline/Gulr-cli)

This is the hands-on manual for Gurl-cli.
Short, sharp, and with just enough detail to get you chaining HTTP and gRPC requests without going insane.

---

## 🚀 Quick Commands

```bash
# Generate default config
go run main.go --config-create

# Generate with custom name
go run main.go --config-create --config=myconfig

# Generate gRPC config  
go run main.go --config-create --config=myconfig --type=grpc

# Generate mixed config
go run main.go --config-create --config=myconfig --type=mixed

# Run config
go run main.go --config=myconfig.json
```

---

## ⚙️ Config Structure

### HTTP Config
```json
{
  "id": "1",
  "type": "http", 
  "url": "http://example.com/api",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json",
    "Authorization": "Bearer {RESPONSE id=1 json:token}"
  },
  "body": {
    "user_id": "{RESPONSE id=2 json:id}"
  },
  "response": "-"
}
```

### gRPC Config
```json
{
  "id": "2",
  "type": "grpc",
  "target": "localhost:50051",
  "endpoint": "auth.AuthService/Login",
  "data": {
    "token": "{RESPONSE id=1 json:token}"
  },
  "metadata": {
    "authorization": "bearer {RESPONSE id=1 json:token}"
  },
  "protofiles": ["/path/to/proto/files"],
  "response": "-"
}
```

> **Note:** `protofiles` and `metadata` are optional. Use absolute paths for protofiles.

### repeated config
```json
{
    "type": "repeated",
    "repeated_id": "6",
    "replace": {
        "Authorization": "Bearer {RESPONSE id=8 json:token}"
        }
    }
}
```

---

## 🔗 Response Placeholders

**Syntax:** `{RESPONSE id=config_id proecssing_type}`

### Supported Locations:
- ✅ **URLs** - `"/api/users/{RESPONSE id=1 json:user_id}"`
- ✅ **Headers** - `"Bearer {RESPONSE id=1 json:token}"`  
- ✅ **Body fields** - `{"id": "{RESPONSE id=2 json:id}"}`
- ✅ **gRPC data** - `{"token": "{RESPONSE id=1 json:token}"}`

### Processing Modes:
- `json:field` - Extract specific field from JSON response
- `none` - Use entire response body

### Examples:

**URL Templating:**
```json
{
  "id": "4",
  "type": "http", 
  "url": "http://localhost:8080/api/users/{RESPONSE id=3 json:user_id}",
  "method": "GET"
}
```

**Header Templating:**
```json
{
  "id": "2",
  "type": "http",
  "headers": {
    "Authorization": "Bearer {RESPONSE id=1 none}"
  }
}
```

---

## 🔄 Repeated Configs

Duplicate existing configs with modifications:

```json
{
  "type": "repeated",
  "repeated_id": "1",
  "replace": {
    "name": "VIZA",
    "email": "viza@example.com"
  }
}
```

**How it works:**
1. Finds config with `id` matching `repeated_id`
2. Creates a copy with new auto-assigned ID  
3. Applies all fields from `replace` to override original values
4. Executes the modified config

**Example transformation:**
```json
// Original config (id=1)
{
  "id": "1",
  "type": "http",
  "url": "http://localhost:8080/api/users",
  "body": {"name": "ziv", "email": "old@email.com"}
}

// Repeated config creates:
{
  "id": "7", // auto-assigned
  "type": "http", 
  "url": "http://localhost:8080/api/users",
  "body": {"name": "VIZA", "email": "viza@example.com"}
}
```

---

## 💡 Pro Tips

1. **Chain everything** - Use responses from any step in URLs, headers, or bodies
2. **Repeated configs are powerful** - Test different payloads without duplication
3. **IDs are automatic** - Don't manually set IDs, the system manages them
4. **Mix and match** - HTTP → gRPC → HTTP chains work seamlessly

---

## 🎯 Real World Example

```json
[
  {
    "id": "1",
    "type": "http",
    "url": "http://localhost:8443/api/users/reg",
    "method": "POST",
    "body": {
      "name": "ziv",
      "email": "test@example.com",
      "password": "secret"
    }
  },
  {
    "id": "2", 
    "type": "http",
    "url": "http://localhost:8443/api/users/log",
    "method": "POST", 
    "body": {
      "name": "ziv",
      "password": "secret"
    }
  },
  {
    "id": "3",
    "type": "http", 
    "url": "http://localhost:8443/api/users/extUserId/{RESPONSE id=1 json:token}",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {RESPONSE id=2 json:token}"
    }
  },
  {
    "type": "repeated",
    "repeated_id": "1",
    "replace": {
      "name": "VIZA",
      "email": "viza@example.com" 
    }
  }
]
```
