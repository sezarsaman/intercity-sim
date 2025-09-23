# Intercity Ride Dispatch & Pricing Simulator

Demo-ready scaffold to showcase **Go microservices**, **Postgres**, **RabbitMQ**, **Docker**, and **CI**.

- `api-gateway` (HTTP): accepts trip requests from clients
- `trip-service` (HTTP + Postgres + RabbitMQ): persists trips, publishes `trip.requested`, consumes `trip.priced`
- `pricing-service` (HTTP + RabbitMQ): consumes `trip.requested`, computes price, publishes `trip.priced`
- `matching-service` (HTTP + Redis Geo + RabbitMQ): consumes `trip.priced`, finds nearby drivers, publishes `trip.matched`

> Roadmap: Observability (OTel/Prometheus/Grafana/Jaeger), K8s Helm charts, load tests, dashboard UI.

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

- **M3 – Event-Driven (Done):**
  - Added RabbitMQ (topic exchange: `rides.events`)
  - Defined minimal events:
    - `trip.requested` (published by trip-service)
    - `trip.priced` (published by pricing-service, consumed by trip-service)
  - End-to-end flow wired:
    - Client → API Gateway → Trip Service → `trip.requested` → Pricing Service → `trip.priced` → Trip Service update

- **M3b – Matching (Done):**
  - Added Redis Geo + new `matching-service`
  - Defined `trip.matched` event
  - Flow: `trip.priced → matching-service → trip.matched → trip-service`
  - Retry/DLQ queues wired for robustness

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
  - `POST /api/v1/trips` → creates trip (triggers event-driven flow)
  - `GET /health`
- **pricing-service** (`:8082`)
  - Consumes `trip.requested`, publishes `trip.priced`
  - `POST /price` (direct compute, for testing)
  - `GET /health`
- **trip-service** (`:8081`)
  - `POST /trips` → create trip
  - `GET /trips/{id}` → fetch by id
  - Consumes `trip.priced` and `trip.matched` to update stored trips
  - `GET /health`
- **matching-service** (`:8083`)
  - Consumes `trip.priced`, publishes `trip.matched`
  - Uses Redis Geo to pick drivers
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
- `ghcr.io/<org-or-user>/intercity-sim/matching-service:<tag>`

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
  matching-service/
pkg/
  events/             # shared event contracts (TripRequested, TripPriced, TripMatched, Envelope)
  mq/                 # RabbitMQ publisher/subscriber abstractions
docker-compose.dev.yml
.github/workflows/ci.yml
go.mod / go.sum
Makefile
```

---

## Next

- Extend matching logic (register/unregister drivers, smarter selection)
- Add observability stack (Prometheus, Grafana, Jaeger with OpenTelemetry)
- Helm charts + K8s deploy
- Retry/DLQ strategies for robustness
- Load tests + simple dashboard UI