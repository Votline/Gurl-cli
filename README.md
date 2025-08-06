# Gurl-cli 🚀

**Supercharged curl/grpcurl with config chaining**  
Stop memorizing complex commands - just save them as reusable configs!

## Features

- ✨ **Config-driven requests** (`.json`)
- ⛓️ **Chain requests** (http-to-http, grpc-to-grpc, or mixed)
- 🔥 **Supports both http AND grpc**
- 🛠️ **Quick config creation** (pre-made templates for both http and grpc)

## Why Gurl?

Because typing **this sucks**:
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '{"param":"value"}' \
  https://api.example.com/endpoint
```
With Gurl, **just**:
```bash
gurl --config=/path/to/my_config.json
```

## How It Works

Gurl-cli is a pure Go tool that:
- Uses `net/http` for HTTP requests
- Uses `google.golang.org/grpc` for gRPC calls
- No external dependencies (curl/grpcurl not needed)
- All configs → actual Go HTTP/gRPC calls

## Quick Start
1. Generate config
2. Edit the config with your actual values
3. Run it:
```bash
gurl --config=/path/to/config.json
```

## Generating configurations
Configurations are saved in the current working directory by default. You can specify a custom path.

#### Basic Usage
```bash
# Generate default HTTP config (config.json)
gurl --config-create

# Custom HTTP config (auth_request.json)
gurl --config-create --config=/path/to/auth_request

# Generate gRPC config (user_service_lookup.json)
gurl --config-create --config=/path/to/user_service_lookup --type=grpc 

# Generate mixed config (auth_then_api.json)
gurl --config-create --config=/path/to/auth_then_api --type=mixed
```

### For HTTP requests (curl-style):
```bash
gurl --config-create --config=config
```
This generates config.json:
```json
{
  "id": "1",
  "type": "http",
  "url": "-",
  "method": "-",
  "headers": {
    "Authorization": "Bearer -",
    "Content-Type": "application/json"
  },
  "data": {}
}
```

### For gRPC requests:
```bash
gurl --config-create --type=grpc --config=grpc_config
```
This generates grpc_config.json:
```json
{
  "id": "1",
  "type": "grpc",
  "endpoint": "service.Method",
  "data": {},
  "metadata": {
    "authorization": "bearer -"
  }
}
```

### For Mixed requests:
```bash
gurl --config-create --type=mixed --config=mixed_config
```
This generates mixed_config.json:
```json
[
  {
    "id": "1",
    "type": "http",
    "url": "-",
    "method": "-",
    "headers": {
      "Content-Type": "application/json"
    },
    "data": {}
  },
  {
    "id": "2",
    "type": "grpc",
    "endpoint": "-",
    "data": {},
    "metadata": {
      "authorization": "bearer -"
    }
  }
]
```
