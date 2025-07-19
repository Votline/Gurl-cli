# Gurl-cli 🚀

**Supercharged curl/grpcurl with config chaining**  
Stop memorizing complex commands - just save them as reusable configs!

## Features

- ✨ **Config-driven requests** (`.json`)
- ⛓️ **Chain requests** (curl-to-curl, grpc-to-grpc, or mixed)
- 🔥 **Supports both curl AND grpcurl**
- 🛠️ **Quick config creation** (pre-made templates for both curl and grpc)

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
gurl --config=my_config.json
```

## How it works
1. Gurl-cli **generates proper commands** based on your `.json` configs
2. **Executes** them using system's:
   - `curl` for HTTP requests
   - `grpcurl` for gRPC calls

## Requirements

**Essential tools** (must be installed separately):
- [`curl`] - for HTTP requests
- [`grpcurl`] - for gRPC requests  

## Quick Start
1. Generate config
2. Edit the config with your actual values
3. Run it:
```bash
gurl --config=config.json
```

## Generating configurations
Configurations are saved in the current working directory by default. You can specify a custom path.

#### Basic Usage
```bash
# Generate default HTTP config (config.json)
gurl --config-create

# Custom HTTP config (auth_request.json)
gurl --config-create --name=~/path/to/auth_request

# Generate gRPC config (user_service_lookup.json)
gurl --config-create --type=grpc --name=user_service_lookup

# Generate mixed config (auth_then_api.json)
gurl --config-create --type=mixed --name=auth_then_api
```

### For HTTP requests (curl-style):
```bash
gurl --config-create --name=config
```
This generates config.json:
```json
{
  "type": "http",
  "url": "-",
  "method": "-",
  "headers": {
    "Authorization": "Bearer -",
    "Content-Type": "application/json"
  }
}
```

### For gRPC requests:
```bash
gurl --config-create --type=grpc --name=grpc_config
```
This generates grpc_config.json:
```json
{
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
gurl --config-create --type=mixed --name=mixed_config
```
This generates mixed_config.json:
```json
[
  {
    "type": "http",
    "url": "-",
    "method": "-",
    "headers": {
      "Content-Type": "application/json"
    }
  },
  {
    "type": "grpc",
    "endpoint": "-",
    "data": {},
    "metadata": {
      "authorization": "bearer -"
    }
  }
]
```
