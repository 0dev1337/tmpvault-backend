# tmpvault-backend

HTTP API for short-lived file uploads: store files on disk, track metadata in MongoDB, serve downloads by ID, and remove expired files in the background.

## Requirements

- [Go](https://go.dev/dl/) 1.25+
- [MongoDB](https://www.mongodb.com/try/download/community) reachable from the machine running the server

## Configuration

The server reads **`config.toml`** from the **process working directory** (the directory you run `go run` or the binary from).

| Section | Keys | Purpose |
|--------|------|---------|
| `[server]` | `host`, `port`, `max_upload_limit` | Listen address and max multipart body size (**MiB**). |
| `[storage]` | `base_path` | Root directory for saved uploads (subfolders per expiry bucket). |
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

**Errors:** JSON body with `status` and `error` (e.g. missing file, unknown expiry, save failures).

### Download a file

- **Method / path:** `GET /download/:file_id`
- **Success (200):** file bytes (`SendFile`).
- **Not found (404):** JSON `{"status":"not found","error":"File not found"}` if the ID is unknown.

## Behavior notes

- **Cleanup:** A background worker runs every **30 seconds**, deletes files whose `expiry` (Unix timestamp) is in the past, and removes the corresponding MongoDB document.
- **Logging:** Structured logs via [zerolog](https://github.com/rs/zerolog); HTTP request logging is applied as middleware.

## Project layout

| Path | Role |
|------|------|
| `cmd/server` | Application entrypoint |
| `internal/config` | TOML config loading |
| `internal/database` | Upload record types and MongoDB access |
| `internal/routes` | Fiber handlers (upload / download) |
| `internal/service` | Expired upload cleanup worker |
| `pkg/mongo` | Shared MongoDB client and collection setup |
| `pkg/logging` | Logger setup |

## License

See the repository’s license file if present.
