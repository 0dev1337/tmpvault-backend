# tmpvault-backend

HTTP API for short-lived file uploads: store files on disk, track metadata in MongoDB, serve downloads by ID, and remove expired files in the background.

## Requirements

- [Go](https://go.dev/dl/) 1.25+
- [MongoDB](https://www.mongodb.com/try/download/community) reachable from the machine running the server

## Configuration

The server reads **`config.toml`** from the **process working directory** (the directory you run `go run` or the binary from).

| Section | Keys | Purpose |
|--------|------|---------|
| `[server]` | `host`, `port`, `max_upload_limit` | Listen address and max total upload size, enforced both as the Fiber `BodyLimit` and as the declared-size ceiling for chunked uploads (**MiB**). |
| `[server]` | `read_timeout_seconds`, `write_timeout_seconds`, `idle_timeout_seconds` | Fiber server timeouts (seconds). `read_timeout_seconds` bounds how long a single request, including its body, may take to arrive — set generously for large direct uploads. Omitted/zero falls back to 900 / 60 / 60 respectively. |
| `[server]` | `chunk_size_mb` | Chunk size (**MiB**) the server hands back from `/api/v1/upload/init` for the chunked upload flow. Defaults to **40** if unset. Keep this under any reverse proxy's per-request body cap sitting in front of this server (e.g. Cloudflare's 100MB limit on Free/Pro/Business plans). |
| `[storage]` | `base_path` | Root directory for saved uploads (subfolders per expiry bucket, plus a `.incoming` scratch dir for in-progress uploads). |
| `[expiry_paths]` | e.g. `"24h" = "1d"` | Maps the **form field** `expiry` to a subdirectory name under `base_path`. Only keys listed here are accepted on upload. |
| `[database]` | `mongodb_url`, `db_name` | MongoDB connection URI and database name. Collection used: **`uploads`**. |

See the checked-in [`config.toml`](config.toml) for an example.

## Run

From the repository root (so `config.toml` is found):

```bash
go run ./cmd/server
```

Or build and run:

```bash
go build -o tmpvault ./cmd/server
./tmpvault
```

Ensure MongoDB is running and matches `mongodb_url` before starting.

## API

Base URL examples below assume `host = "0.0.0.0"` and `port = 8080`.

### Upload a file

- **Method / path:** `POST /api/v1/upload`
- **Content-Type:** `multipart/form-data`
- **Fields:**
  - **`file`** (required): the file to store.
  - **`expiry`** (optional): must match a key in `[expiry_paths]`; defaults to **`24h`** if omitted.
  - **`file_name_length`** (optional): random-name length before extension (e.g. `8` -> `Ab12Cd34.csv`), must be a positive integer.

**Success (200):**

```json
{
  "id": "<generated-file-id>",
  "expiry": "24h",
  "status": "success"
}
```

Use `id` as `file_id` when downloading.

**Errors:** JSON body with `status` and `error` (e.g. missing file, unknown expiry, insufficient storage, save failures).

Use this endpoint for files comfortably under any reverse proxy's per-request body cap in front of this server (e.g. Cloudflare's plan-based limit, commonly 100MB). For anything larger, or any upload that should survive a transient network blip, use the chunked flow below instead.

### Upload a file (chunked)

Three calls: start a session, send every chunk, then finalize. The chunked and direct paths converge on the exact same dedupe/save logic and produce an identically-shaped success response.

#### 1. Start a session

- **Method / path:** `POST /api/v1/upload/init`
- **Content-Type:** `application/x-www-form-urlencoded` or `multipart/form-data`
- **Fields:**
  - **`size`** (required): total file size in bytes. Rejected if it exceeds `max_upload_limit`.
  - **`filename`** (required): original filename (used for the stored extension and the record's display name).
  - **`expiry`** (optional): same as the direct endpoint; defaults to `24h`.
  - **`file_name_length`** (optional): same as the direct endpoint.

**Success (200):**

```json
{
  "status": "success",
  "session_id": "<opaque session id>",
  "chunk_size": 41943040,
  "total_chunks": 4
}
```

`chunk_size` is in **bytes**. The client must use exactly this size for every chunk except the last, which is whatever remains (`size - chunk_size * (total_chunks - 1)`).

#### 2. Upload each chunk

- **Method / path:** `POST /api/v1/upload/{session_id}/chunk?index=N`
- **Content-Type:** `application/octet-stream` — the raw chunk bytes as the request body, **not** multipart.
- **`index`**: 0-based chunk number. Chunks may be sent in any order and safely retried (each is written to its own fixed byte offset in the session's temp file).
- **Success (200):** `{"status": "success"}`
- **Errors (400):** wrong chunk length for that index, invalid/out-of-range index. **(404):** unknown or expired session.

#### 3. Complete the upload

- **Method / path:** `POST /api/v1/upload/{session_id}/complete`
- **Success (200):** same shape as the direct endpoint — `{"id": "...", "expiry": "...", "status": "success"}`.
- **Errors (400):** one or more chunks never arrived (`{"error": "missing chunk N"}`). **(404):** unknown or expired session.

Sessions live in memory only and are abandoned (temp file removed) after **2 hours** of inactivity; a server restart also drops any upload in progress, so the client should treat either case as a failure and restart from `/init`.

### Download a file

- **Method / path:** `GET /download/:file_id`
- **Success (200):** file bytes (`SendFile`).
- **Not found (404):** JSON `{"status":"not found","error":"File not found"}` if the ID is unknown.

## Behavior notes

- **Cleanup:** A background worker runs every **30 seconds**, deletes files whose `expiry` (Unix timestamp) is in the past, and removes the corresponding MongoDB document.
- **Chunked upload sessions:** A separate janitor runs every **5 minutes** and drops any chunked-upload session (and its temp file) idle for more than 2 hours.
- **Disk space:** Both upload paths check free disk space on `base_path` before accepting the declared/actual file size and return **507** if there isn't enough room, rather than failing partway through a long write.
- **Deduplication:** Uploads are fingerprinted by SHA-256; a second upload of identical content (while the first hasn't expired) reuses the existing file instead of storing a duplicate.
- **Logging:** Structured logs via [zerolog](https://github.com/rs/zerolog); HTTP request logging is applied as middleware.

## Project layout

| Path | Role |
|------|------|
| `cmd/server` | Application entrypoint |
| `internal/config` | TOML config loading |
| `internal/database` | Upload record types and MongoDB access |
| `internal/routes` | Fiber handlers (upload / chunked upload / download) |
| `internal/service` | Expired upload cleanup worker |
| `pkg/mongo` | Shared MongoDB client and collection setup |
| `pkg/logging` | Logger setup |

## License

See the repository’s license file if present.
