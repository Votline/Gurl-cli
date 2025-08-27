# Gurl-cli GUIDE 📖

This is the **hands-on manual** for Gurl-cli.  
Short, sharp, and with just enough detail to get you chaining HTTP and gRPC requests without going insane.

---

## Commands

There are only a few flags. No bullshit.

```bash
# Generate default HTTP config
go run main.go --config-create

# Generate HTTP config with custom name (myconfig.json)
go run main.go --config-create --config=myconfig

# Generate gRPC config
go run main.go --config-create --config=myconfig --type=grpc

# Generate mixed config (HTTP + gRPC)
go run main.go --config-create --config=myconfig --type=mixed

# Run any config
go run main.go --config=myconfig.json
````

That’s it. No hidden switches, no 50-page manual.

---

## Configs

Configs are just JSON.
You edit them → run them → chain them. Done.

### HTTP Config

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
  "body": {},
  "response": "-"
}
```

### gRPC Config

```json
{
  "id": "1",
  "type": "grpc",
  "target": "-",
  "endpoint": "service.Method",
  "data": {},
  "metadata": {
    "authorization": "bearer -"
  },
  "response": "-",
  "protofiles": [
    "-"
  ]
}
```

Notes:

* **`protofiles`** is optional → if missing, Gurl tries to discover services with `protoreflect`.
* **`metadata`** is optional → nuke it if you don’t care.
* If you do specify `protofiles`, use **absolute paths**.

### Mixed Config (HTTP + gRPC chain)

```json
[
  {
    "id": "1",
    "type": "http",
    "url": "-",
    "method": "-",
    "headers": {
      "Authorization": "Bearer -",
      "Content-Type": "application/json"
    },
    "body": {},
    "response": "-"
  },
  {
    "id": "2",
    "type": "grpc",
    "target": "-",
    "endpoint": "service.Method",
    "data": {},
    "metadata": {
      "authorization": "bearer -"
    },
    "response": "-",
    "protofiles": [
      "-"
    ]
  }
]
```

(Same rules apply: `protofiles` + `metadata` are optional in the gRPC parts.)

---

## Instructions (a.k.a. Response Placeholders)

The magic sauce for chaining requests is **`{RESPONSE ...}`**.
You drop it anywhere inside your configs, and it gets replaced at runtime.

```bash
{RESPONSE id=cfgID key:value}
```

Rules:

1. **Must** be wrapped in `{}`.
2. **Must** specify the config `id`.
3. **Must** specify the processing mode:

   * `json:field` → extracts a field from the response JSON.
   * `none` → dumps the entire response into place.
4. Works everywhere: HTTP headers, bodies, gRPC data… doesn’t matter.

### Example

Use token from step 1 in step 2:

```json
{
  "id": "2",
  "type": "http",
  "url": "http://localhost:8080/api/todos/task",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer {RESPONSE id=1 json:token}",
    "Content-Type": "application/json"
  },
  "body": {
    "title": "some title",
    "content": "some content"
  },
  "response": "-"
}
```

---

## TL;DR

* **`--config-create`** → makes a starter config.
* Edit JSON → add your endpoints, methods, headers, data.
* Use **`{RESPONSE ...}`** to chain results.
* Run with **`--config`**.

That’s the whole game. 🚀

