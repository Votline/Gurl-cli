# README.md

# 🚀 Gurl-cli

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Zero--Allocation-Optimized-orange?style=for-the-badge" alt="Performance">
  <img src="https://img.shields.io/badge/gRPC-Supported-blue?style=for-the-badge" alt="gRPC">
  <img src="https://img.shields.io/badge/WebSockets-Supported-purple?style=for-the-badge" alt="WebSockets">
</p>

**Gurl-cli** is a high-performance, stateful HTTP, gRPC & WebSocket client designed for the command line.

Unlike standard tools (`curl`, `postman`, `grpcurl`, `websocat`), `gurl-cli` is engineered for **request chaining** and automated flow testing. It allows you to execute sequences of requests where the output of one (e.g., Auth Tokens, session cookies, generated UUIDs) automatically feeds into the next. Everything is defined in a single, readable file using a custom, zero-allocation configuration format (`.gurlf`) that completely eliminates JSON escaping hell.

---

## 🗺 Navigation

* [⚡ Quickstart](#-quickstart)
* [🛠 Installation](#-installation)
* [📄 Configuration Types](#-configuration-types)
  * [1. HTTP](#1-http-stateful-requests)
  * [2. gRPC](#2-grpc-first-class-support)
  * [3. WebSockets](#3-websockets-real-time-flows)
  * [4. Repeat](#4-repeat-dry-principle)
  * [5. Import](#5-import-modular-configs)
* [🧩 Dynamic Macros & Flow Control](#-dynamic-macros--flow-control)
* [🧪 Integration Testing](#-integration-testing)
* [⚠️ Core Concepts & Constraints](#warning-core-concepts--constraints)
* [🏗 Architecture & Performance](#-architecture--performance)
* [📜 License](#-license)

---

## ⚡ Quickstart

`gurl-cli` operates via a clean command-line interface, allowing you to run files or raw configurations directly.

```bash
# Run a specific configuration file
gurl-cli run config.gurlf

# Run a raw configuration directly from the terminal (no file required)
gurl-cli run "
[http_config]
URL:http://localhost:8080
Type:http
ID:0
[\http_config]"

# Create a template or get help
gurl-cli create config.gurlf http
gurl-cli help
```

---

## 🛠 Installation

Ensure you have Go installed, then run:

```bash
go install https://github.com/Votline/Gurl-cli@latest
```

Or download from github [Releases](https://github.com/Votline/Gurl-cli/releases)

---

## 📄 Configuration Types

Configs use the custom `.gurlf` syntax. The tool writes responses **back into the configuration file** automatically, making debugging instant.

### 1. HTTP (Stateful Requests)

Standard HTTP requests support automatic cookie jars, headers, and body payloads.

```text
[reg]
URL:http://localhost:8080/api/users/reg
Method:POST
Body:`
    {
        "name": "Viza",
        "email": "some@mail.com",
        "role":"admin",
        "password":"eightpswd"
    }
`
Headers:Content-Type: application/json
ID:0
Type:http
[\reg]
```

### 2. gRPC (First-Class Support)

Easily test your microservices by pointing directly to your `.proto` files or using reflection inside the grpc of your servers.

```text
[new_course]
Target:localhost:50052
Endpoint:courses.CoursesService/NewCourse
Data:`
    {
        "user_id": "12345",
        "name": "some_name",
        "description": "cool desc",
        "price": "1347"
    }
`
ProtoPath:../protos/courses.proto
ID:0
Type:grpc
[\new_course]
```

### 3. WebSockets (Real-Time Flows)

Full support for interactive WebSocket connections natively through the HTTP type.
* Use `ws://` to send a single configured payload.
* Use `while:ws://` to open an interactive session and stream messages directly from your terminal's `stdin`.

```text
[chat_session]
URL:while:ws://localhost:8080/ws
ID:1
Type:http
[\chat_session]
```

### 4. Repeat (DRY Principle)

Don't rewrite identical payloads. Inherit from a previous request using `TargetID` and patch only the fields you need using `Replace`.

```text
[log_upd_user]
TargetID:1
Replace:`
[rep]
Body:`
    {
        "name": "a62893bc-70af-4fb3-aca6-b663ad35404f",
        "email": "upd@mail.com",
        "password": "updatepaswd"
    }`
[\rep]
`
ID:8
Type:repeat
[\log_upd_user]
```

### 5. Import (Modular Configs)

Keep your configs clean by importing base templates or fallback configurations. Variables can be passed down to the imported scope.

```text
[import_config]
TargetPath:fallback.gurlf
ID:2
Type:import
SetVariables:`
[vars]
UserID: FALLBACK {RANDOM oneof=uuid}
[\vars]
`
[\import_config]
```

---

## 🧩 Dynamic Macros & Flow Control

`gurl-cli` becomes powerful when you link requests together using in-place macros.

### 1. Response & Cookie Injection
Extract data from previous requests or stateful fields:
* **JSON Extraction:** `{RESPONSE id=1 json:token}` - Extracts a field from the response body of config `ID:1`.
* **Stateful Cookies:** Use the `CookieIn` field to inject session data:
    * `{COOKIES id=1}` - Injects all cookies captured in the `CookieOut` field of config `ID:1`.
    * `{COOKIES id=file}` - Tells the transport to use cookies defined directly within the current `CookieIn` block.

### 2. Environment Integration
Manage state across different .gurlf files or system sessions. This is the heavy lifting for cross-file communication.
* **Save State:** Use `SetEnvironments` to persist values to a local `.env_temp` file or OS env.
* **Retrieve State:** `{ENVIRONMENT key=Token ; from=.env_temp}` or `{ENVIRONMENT key=USER ; from=os}`.
* **Defaults:** You can now provide fallback values using `; default=...` if the environment or variable is missing.

> Example
> ```text
> [config]
> SetEnvironments:`
> [envs] <- there is no such file, the environment will be in-memory.
> OsEnvName:os env {RANDOM oneof=uuid}
> [\envs]
> [.env] <- this file exists. The old values will be overwritten by the new ones in case of a collision.
> FileEnvName:file env {RANDOM oneof=uuid}
> [\.env]
> `
> Body:`
>   {
>     "host": "{ENVIRONMENT key=DB_HOST ; from=os ; default=localhost}" <- will be used 'localhost'
>     "osname": "{ENVIRONMENT key=OsEnvName ; from=os ; default=nop}" <- will be used value from os.GetEnv
>     "filename": "{ENVIRONMENT key=FileEnvName ; from=.env ; default=nop}" <- will be used value from file '.env'
>   }
> `
> [\config]
> ```

### 3. Variables Integration
Variables are designed for in-memory state management during a single execution. They do not persist to disk.
* **In-Place Usage:** `{VARIABLE key=Name ; default=guest}` - Get a variable with a fallback.
* **Import config Pattern:** The primary use case is passing data into import configs. 
> Example
> ```text
> [config]
> TargetPath: fallback.gurlf
> SetVariables:`
> [vars]
> UserID: FALLBACK - {RANDOM oneof=uuid}
> [\vars]
> `
> Type:import
> [\config]
> ```
> Note: `fallback.gurlf` will now be able to resolve `{VARIABLE key=UserID}`.

### 4. Smart Randomization
Generate dynamic data in `0 allocs/op`:
* `{RANDOM oneof=uuid}` - High-speed UUID.
* `{RANDOM oneof=user,admin}` - Random pick from a list.
* `{RANDOM oneof=int(10,100)}` - Random integer in range.
* `{RANDOM oneof=int}` - Random integer.

### 5. Flow Control & Failures
* **Wait:** `Wait: 5s` - Delays execution (supports `ms, s, m, h`).
* **Timeout:** `Timeout: 15s` - Sets a dynamic context deadline for the request (stops hanging connections).
* **Expect:** Define success criteria and branching logic:
    * `Expect: 200;fail=crash` - Hard stop on failure.
    * `Expect: 200;fail=5` - If not 200, jump to config `ID:5` and stop.
    * `Expect: 0` - (gRPC) Expects `OK` status.
 
### 6. Request settings
* **The `IgnoreCert` Field:** Adding `IgnoreCert: true` to a config (or inherited via repeat) will skip TLS/SSL verification for that request. Use `false` or omit to enforce secure connections.
* **Context Timeouts:** The `Timeout` field sets a timeout for the context. Automatically inherited by child `repeat` configs.

---

## 🧪 Integration Testing

Because `gurl-cli` persists its state (Responses, Cookies, Envs) back into files, it is perfect for complex integration scenarios.

### Scenario: Cross-file state persistence
You can run a chain of requests where `file2.gurlf` depends on the output of `file1.gurlf`.

**Step 1: `auth.gurlf`**
Captures cookies and saves the `UserID` to a temporary environment.
```text
[login]
URL: http://localhost:8080/login
SetEnvironments: `
    [.env_temp]
    UserID: {RESPONSE id=0 json:id}
    Cookies: "{COOKIES id=0}"
    [\.env_temp]
`
ID: 0
Type: http
[\login]
```

**Step 2: `delete_account.gurlf`**
Uses the previously saved cookies and ID.
```text
[del]
URL: http://localhost:8080/user/{ENVIRONMENT key=UserID ; from=.env_temp}
CookieIn: `
    {COOKIES id=file}
    {ENVIRONMENT key=Cookies ; from=.env_temp}
`
Type: http
[\del]
```

**Step 3: `check.gurlf`**
Combine two files into one for easy startup.
```bash
[import_login]
TargetPath:auth.gurlf
ID:0
Type:import
[\import_login]

[import_delete]
TargetPath:delete_account.gurlf
ID:1
Type:import
[\import_delete]
```

**Step 4: run it**
```bash
gcli run check.gurlf
```

---

## :warning: Core Concepts & Constraints

### 1. The Backtick & Newline Rule (Nesting)
The `.gurlf` parser uses a high-speed, zero-allocation scanning logic. It identifies the end of a multi-line field by looking for a backtick strictly surrounded by newlines (`\n` + \` + `\n`).

> [!CAUTION]
> **A backtick on a new line always terminates the top-level block.** When nesting configs, ensure the inner backticks do not mimic this pattern.

* **CORRECT:** Keep the inner closing backtick on the same line as your data.
* **INCORRECT:** Putting the inner closing backtick on a new line will break the outer parser.

| Syntax | Result |
| :--- | :--- |
| ``Body: `value` `` | One-line string. Value is `value` |
| ``Body: "value\n"`` | Multi-line string. Values is `"value\n"` |
| ``Replace: `[patch]\n Body:`data`\n[\patch]\n`\n `` | **Nested Correct**. The last backtick is surrounded by `\n`, which means the end of the `Replace` key value |
| ``Replace: `[patch]\n Body:`data\n`\n [\patch]\n`\n `` | **Nested Incorrect**. There is a `\n`\n` in the value, which means the end of the value for the 'Replace' field, but this is not the case. |

> [!NOTE]
> Correct config:
> ```text
> [config]
> GlobalKey:`
> [inner_cfg]
> LocalKey:`
> LocalValue` <- end of 'LocalKey'. Syntax: '`\n'
> [\inner_cfg]
> ` <- end of 'GlobalKey'. Syntax: '\n`\n'
> [config]


### 2. Configuration Field Rules

* **The `Replace` Field:** Strictly exclusive to the `repeat` configuration type. Used to patch fields from a `TargetID`.
* **gRPC Reflection vs. ProtoPath:** Omit `ProtoPath` to use Server Reflection. If provided, it parses the local `.proto` file.


### 3. Syntax Upgrades
Macros now use `;` as a separator to allow for clean default fallbacks (e.g., `{ENVIRONMENT key=API_KEY ; from=os ; default=12345}`). You can escape semicolons inside values if needed.

---

## 🏗 Architecture & Performance

`gurl-cli` is explicitly engineered for **low latency and zero-allocation execution**, making it viable for high-throughput testing and microservice orchestration. The core architecture relies on aggressive object pooling and deep Go runtime optimizations.

### 1. Pre-allocation & Interface Hacking
Interface Hacking & Pre-allocation: To avoid heap allocations, the config package uses pre-allocated pools (10 objects per type). The parser peeks at the Type field and uses unsafe.Pointer to manually swap itab and data pointers.

### 2. Lock-Free Pipeline (Ring Buffers)
Lock-Free Pipeline: Data flows between the Parser, Core, File Writer and Console Writer via isolated, capped Ring Buffers. This decoupled architecture ensures the executor never blocks on the parser or disk I/O.

### 3. Optimized Custom Tooling:
Standard library functions (bytes.Split, json, strconv) are replaced with custom, allocation-free iterators. fastUUID and manual byte-slice scanning keep the hot path garbage-free.

### 4. Resilient Disk I/O:
The file writer uses a detached seek/flush mechanism to inject responses directly into .gurlf files. This context-free operation prevents file corruption even during abrupt process termination.

### ⚡ Benchmarks
Tested on **AMD Ryzen 7 5800U** (`linux/amd64`). The hot paths are garbage-free.

| Component / Function | Time (ns/op) | Allocations | Notes |
| :--- | :--- | :--- | :--- |
| **ParseStream** | **~362 ns/op** | **0 allocs/op** | Full multi-line config streaming. |
| **HandleType** | **~274 ns/op** | **0 allocs/op** | `unsafe` interface casting + mapping. |
| **HandleInstructions** | **~106 ns/op** | **0 allocs/op** | Finds macros (aka instructions) |
| **ParseFindConfig** | **~317 ns/op** | **0 allocs/op** | Finds config by target id, anmarshall it (import configs logic) |
| **fastExtract** | **~12.6 ns/op** | **0 allocs/op** | Rapid field extraction. |
| **fastUUID** | **~48 ns/op** | **0 allocs/op** | Generate UUID (libraries allocate memory) |
> for 0 alloc in ParseStream, you need to comment out log.Debug

---

## 📜 License

  - **License:** This project is licensed under [MIT](https://www.google.com/search?q=LICENSE)
  - **Third-party Licenses:** Third-party [licenses/](https://www.google.com/search?q=licenses/).

