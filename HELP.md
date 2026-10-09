# Sortener — End to End Guide

A complete guide from cloning the repository to getting the server running and passing tests.
---

## 1. Clone repository

```bash
git clone git@github.com:awinishere/Sortener.git
cd Sortener
```

## 2. Install Go SDK matching the version in `go.mod`

The required version is always specified on the `go` line in `go.mod` (currently `1.26.9`).

**Linux:**
```bash
curl -LO https://go.dev/dl/go1.26.9.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.9.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin       
go version                                
```

**Other valid alternatives:**
- **IntelliJ IDEA**: *Settings → Go → GOROOT* point to your Go installation, or let the IDE download the SDK automatically.

- **`GOTOOLCHAIN=auto` (default)**: If your installed toolchain is older (e.g., 1.26.5), running `go run/go build/go test` commands in this folder will **automatically download** the matching version specified in `go.mod`. If you prefer this approach, you can skip the installation step above.

- **Windows/macOS**: Official `.msi`/`.pkg` installers from <https://go.dev/dl/>.

> Note: The version must be ≥ the version specified in `go.mod`. Downgrading the Go version will cause builds to fail with errors such as `go.mod requires go >= 1.26.9`.

## 3. Install project dependencies

```bash
go mod download
go mod verify        # optional: verify module integrity
```

All dependencies (redis client, godotenv, miniredis, swag, http-swagger) are locked in `go.sum`; no extra manual steps are required.

## 4. Prepare configuration (configured via environment variables, NO defaults)

```bash
cp .env.example .env
```

Fill in `.env` (there are **no defaults in the code** for required variables):

| Variable | Required | Description |
|----------|:--------:|-------------|
| `ALPHABET` | ✅ | Characters used for generating short codes |
| `TTL` | ✅ | Link lifetime, Go duration format (`720h`, `30m`, `1h30m`) |
| `REDIS_ADDR` | ✅ | `host:port`, `redis://…` or `rediss://…` (TLS) |
| `REDIS_USERNAME` / `REDIS_PASSWORD` | — | Redis credentials/ACL (ignored when already embedded in the URL) |
| `BASE_URL` | — | Public base URL used in the `short_url` field (default `http://localhost:8080`) |
| `PORT` | — | HTTP server listen address (default `:8080`) |

> The code is **fail-fast**: if any required variable is empty or invalid, the application terminates immediately with a clear error message. The `.env` file is never packaged into Docker images (listed in `.dockerignore`) — in containers, pass these values via environment variables instead.

**Local Redis (if not using cloud):**
```bash
docker compose up -d store     # Redis on localhost:6379, persistent via volume
```

## 5. Run unit tests

```bash
go vet ./...               # static check, must pass clean
go test -race -count=1 ./...   # run all tests (uses miniredis, no real Redis required)
```

Test coverage includes: `cmd/store_test.go` (config, encode, Store + TTL + expiry), `cmd/handler_test.go` (validation & response handler), `cmd/router_test.go` (`/api/v1` routing, 404/405, swagger).

## 6. Run the server

```bash
go run ./cmd
```

Expected startup logs:
```
no .env file found...        ← only appears if no .env file exists in the working directory
running on :8080
```

**Endpoint smoke test:**
```bash
# create short URL
curl -X POST http://localhost:8080/api/v1/shorten \
  -d '{"url":"https://example.com/some/long/path"}'
# → {"code":"1","short_url":"http://localhost:8080/api/v1/1"}

# redirect to original URL
curl -i http://localhost:8080/api/v1/1        # → 302 + Location header

# API documentation (Swagger UI)
open http://localhost:8080/api/v1/swagger/index.html
```

Every incoming request will be printed to the logs: `INFO request method=POST path=/api/v1/shorten status=201 dur=...`.

## 7. Regenerate Swagger docs (only when API annotations change)

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/main.go -o docs
```

## 8. CI/CD (GitHub Actions) — automated on push

| Workflow | Trigger | Action |
|----------|---------|-----|
| `deps.yml` | `go.mod`/`go.sum` changes, weekly | check outdated modules (warning only) + `govulncheck` (fails on vulnerabilities) |
| `test.yml` | push to master / PR | `go vet` + `go test -race` |
| `deploy.yml` | after test passes on master | build image → push to GHCR → trigger Render deploy hook |

Requires the `RENDER_HOOK_URL` repository secret to trigger deployments on Render (see `deploy.yml`).

---

## Troubleshooting

| Symptom | Cause & Solution |
|--------|-------------------|
| `ALPHABET is required` / `TTL is required` | `.env` file not created/populated, or executed from a different working directory |
| `config error: invalid TTL "720"` | `TTL` must include time units (`720h`), not plain numbers |
| `cannot reach redis` | Incorrect `REDIS_ADDR` / Cloud Redis requires TLS → use `rediss://` protocol |
| `WRONGPASS invalid username-password pair` | Incorrect `REDIS_USERNAME` or `REDIS_PASSWORD` |
| `listen tcp :8080: bind: address already in use` | Old server instance still running — terminate it first, or set `PORT=:8081` |
| `go.mod requires go >= 1.26.9` | Outdated Go version — upgrade your toolchain or rely on `GOTOOLCHAIN=auto` |
| `encode` panic division by zero | Unreachable with current fail-fast config, but if it happens: `LoadConfig()` was not invoked |