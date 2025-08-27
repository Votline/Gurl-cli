# Gurl-cli 🚀

**Supercharged curl/grpcurl with config chaining.**  
Stop memorizing complex flags — save them as small JSON configs and hit run.

---

## Why this exists

Typing long curl/grpcurl incantations sucks. Gurl turns them into tiny, reusable configs you can chain:
- HTTP → HTTP, gRPC → gRPC, or mix them.
- No external binaries required — pure Go (`net/http` + `google.golang.org/grpc` + `github.com/jhump/protoreflect`).
- Responses from earlier steps can feed later steps via simple placeholders.

---

## Quick Start

```bash
# 1) Generate a starter config (HTTP)
go run main.go --config-create

# 2) Edit it with your values
vim http_config.json

# 3) Run it
go run main.go --config=http_config.json
````

Example `http_config.json` you might start from:

```json
[
  {
    "id": "1",
    "type": "http",
    "url": "http://localhost:8080/api/todos/reg",
    "method": "POST",
    "headers": { "Content-Type": "application/json" },
    "body": {
      "id": "Votline",
      "first_name": "Votl",
      "last_name": "line",
      "password_hash": "123"
    },
    "response": "-"
  }
]
```

> Tip: Use placeholders like `"{RESPONSE id=1 json:token}"` to pipe earlier responses into later requests.

---

## The Meat (copy–paste configs)

### HTTP chain (auth → create → list → update → delete)

```json
[
  {
    "id": "1",
    "type": "http",
    "url": "http://localhost:8080/api/todos/reg",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "id": "Votline",
      "first_name": "Vot",
      "last_name": "line",
      "password_hash": "123"
    },
    "response": "-"
  },
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
      "content": "some content",
      "category_id": "some category",
      "done": false
    },
    "response": "-"
  },
  {
    "id": "3",
    "type": "http",
    "url": "http://localhost:8080/api/todos/task",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {RESPONSE id=1 json:token}",
      "Content-Type": "application/json"
    },
    "body": {
      "title": "some title"
    },
    "response": "-"
  },
  {
    "id": "4",
    "type": "http",
    "url": "http://localhost:8080/api/todos/task",
    "method": "PUT",
    "headers": {
      "Authorization": "Bearer {RESPONSE id=1 json:token}",
      "Content-Type": "application/json"
    },
    "body": {
      "id": "1",
      "title": "no some title"
    },
    "response": "-"
  },
  {
    "id": "5",
    "type": "http",
    "url": "http://localhost:8080/api/todos/task",
    "method": "DELETE",
    "headers": {
      "Authorization": "Bearer {RESPONSE id=1 json:token}",
      "Content-Type": "application/json"
    },
    "body": {
      "id": "1"
    },
    "response": "-"
  }
]
```

### gRPC step (reusing token from step 1)

```json
{
  "id": "6",
  "type": "grpc",
  "target": "localhost:50051",
  "endpoint": "auth.AuthService/ExtUserID",
  "data": {
    "token": "{RESPONSE id=1 json:token}"
  },
  "response": "-"
}
```

---

## Docs

* **Full guide:** see [`GUIDE.md`](GUIDE.md) for detailed schema, chaining, and tips.
* **License:** MIT (see `LICENSE`).
* **Licenses** The full license texts are available in the [licenses directory](licenses/)
