# Intercity Ride Dispatch & Pricing Simulator

Demo-ready scaffold to showcase **Go microservices**, **Postgres**, **Docker**, and **CI**.

- `api-gateway` (HTTP): orchestrates trip creation → calls `pricing-service` → calls `trip-service`
- `pricing-service` (HTTP): computes a simple dynamic price (distance + hour-based surge)
- `trip-service` (HTTP + Postgres): persists trips and exposes basic CRUD

> Roadmap: RabbitMQ, Redis Geo, Observability (OTel/Prometheus/Grafana/Jaeger), K8s Helm charts, load tests, dashboard UI.

---

## Milestones

- **M1 – Scaffold (Done):**
  - Base services (`api-gateway`, `pricing-service`, `trip-service + Postgres`)
  - Multi-stage Dockerfiles, `docker-compose.dev.yml`, `Makefile`
  - CI skeleton for build/test and image build/push

- **M2 – Tests & Lint (Done):**
  - `golangci-lint` wired in CI (errcheck/govet/revive/staticcheck/gocritic via default profile)
  - Unit tests for `api-gateway` (with stubbed `pricing-service`)
  - Integration tests for `trip-service` using **testcontainers-go** (Postgres)
  - Stable GitHub Actions workflow (module download/verify + quick build before lint)

- **M3 – Event-Driven (Next):**
  - Add RabbitMQ + minimal event contracts
  - Flow: `trip.requested → pricing → trip.priced`
  - (Stretch for M3b) matching + Redis Geo + DLQ/retry

---

## Quick Start (Dev)

```bash
# 1) (Optional) Set your module path in go.mod to your repo
#    module github.com/yourname/intercity-sim → module github.com/<you>/intercity-sim

# 2) Start services in dev
make run
# or:
# docker compose -f docker-compose.dev.yml up --build

# 3) Create a trip via the gateway
curl -X POST http://localhost:8080/api/v1/trips   -H "Content-Type: application/json"   -d '{
    "passenger_id":"p-1",
    "origin":{"lat":35.6892,"lng":51.3890,"city":"TEH"},
    "destination":{"lat":36.2605,"lng":59.6168,"city":"MHD"},
    "vehicle_type":"sedan"
  }' | jq

# 4) Find the created trip by id (replace <id>)
curl http://localhost:8081/trips/<id> | jq
```

Stop:
```bash
make stop
# or: docker compose -f docker-compose.dev.yml down -v
```

---

## Services & Endpoints

- **api-gateway** (`:8080`)
  - `POST /api/v1/trips` → returns created trip (includes `quoted_price`)
  - `GET /health`
- **pricing-service** (`:8082`)
  - `POST /price` → returns `{distance_km, base, per_km, surge, final}`
  - `GET /health`
- **trip-service** (`:8081`)
  - `POST /trips` → create
  - `GET /trips/{id}` → fetch by id
  - `GET /health`

---

## Testing & Linting (M2)

Local:
```bash
# Lint (needs golangci-lint installed locally)
golangci-lint run -v --timeout=5m

# Unit tests (all except trip-service integration)
go test -v $(go list ./... | grep -v services/trip-service) -count=1

# Integration tests (trip-service with Testcontainers)
go test -v ./services/trip-service -count=1
```

CI (GitHub Actions):
- Reads Go version from `go.mod`
- `go mod download && go mod verify`
- Quick `go build ./...` before lint to stabilize typecheck
- Runs `golangci-lint`, unit tests, then integration tests
- On success, builds & pushes Docker images to **GHCR** with tags:
  - Branch tag: `latest` (main) / `develop` (develop)
  - Immutable tag: `${{ github.sha }}`

> **GHCR:** Ensure “Actions → Read and write permissions” is enabled in repo settings for Packages.

---

## Docker Images

Multi-stage builds per service. CI pushes:
- `ghcr.io/<org-or-user>/intercity-sim/api-gateway:<tag>`
- `ghcr.io/<org-or-user>/intercity-sim/pricing-service:<tag>`
- `ghcr.io/<org-or-user>/intercity-sim/trip-service:<tag>`

Tips:
- Builder stage downloads modules first for better cache.
- Runtime stage is distroless, non-root.

---

## Project Structure

```
/services/
  api-gateway/
  pricing-service/
  trip-service/
pkg/                     # shared libs (future: events, mq, tracing, config)
docker-compose.dev.yml
.github/workflows/ci.yml
go.mod / go.sum
Makefile
```

---

## Next (M3 – Event-Driven)

- Add RabbitMQ (topic exchange: `rides.events`)
- Define minimal events: `trip.requested`, `trip.priced`
- Implement publisher on `trip-service` (on create), consumer on `pricing-service` (compute & publish), consumer on `trip-service` (persist price)
- Seed observability fields in event headers (`event_id`, `occurred_at`, `trace_id`)