# Go Task API

A small REST API for managing a to-do list, written in Go using only the standard
library. I built it to learn Go's HTTP server, JSON handling, concurrency basics and
testing, coming from a Python (Flask) background.

## Endpoints

| Method | Path | What it does |
| --- | --- | --- |
| GET | `/health` | Returns `{"status":"ok"}` |
| GET | `/tasks` | List tasks. Optional filters: `?done=true` / `?done=false`, and `?q=keyword` (case-insensitive title search). Filters can be combined. |
| POST | `/tasks` | Create a task: `{"title": "..."}` |
| GET | `/tasks/{id}` | Get one task |
| PATCH | `/tasks/{id}` | Update `title` and/or `done` |
| DELETE | `/tasks/{id}` | Delete a task |

Errors come back as JSON, e.g. `{"error":"title is required"}`, with a suitable status
code (400 bad input, 404 not found).

## Run it

You need [Go](https://go.dev/dl/) 1.22 or newer (the routing patterns used here need 1.22).

```bash
go run .
```

The server starts on http://localhost:8080 (set `PORT` to change it).

### Try it

Mac / Linux / Git Bash:

```bash
curl -X POST localhost:8080/tasks -d '{"title":"Learn Go"}'
curl localhost:8080/tasks
curl "localhost:8080/tasks?q=learn"
curl -X PATCH localhost:8080/tasks/1 -d '{"done":true}'
```

Windows PowerShell:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/tasks -Body '{"title":"Learn Go"}'
Invoke-RestMethod http://localhost:8080/tasks
```

## Tests

```bash
go test ./...
go test -race ./...    # also checks the store is safe with concurrent requests
```

## Design notes

- `store.go` holds the data. A mutex protects the map because Go handles each HTTP
  request in its own goroutine.
- `handlers.go` handles routing, validation and JSON responses.
- Request bodies are limited to 1 MB, and unknown JSON fields are rejected.
- Search is a case-insensitive substring match on the title, combined with the `done` filter.
- `PATCH` uses pointer fields so "not sent" can be told apart from "sent as false".

## Limitations and ideas

- Data is held in memory, so it is lost when the server stops. A database (for example
  PostgreSQL) would fix that.
- No authentication yet (JWT would be a natural next step).
- Ideas: persistence, search across more fields, a Dockerfile, pagination, structured logging.
