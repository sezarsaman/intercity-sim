# Intercity Ride Dispatch & Pricing Simulator (Milestone 1)

Demo-ready scaffold to showcase **Go microservices**, **Postgres**, **Docker**, and **CI**:
- `api-gateway` (HTTP): orchestrates trip creation → calls `pricing-service` → calls `trip-service`
- `pricing-service` (HTTP): computes a simple dynamic price based on distance + hour surge
- `trip-service` (HTTP + Postgres): persists trips and exposes basic CRUD

> Roadmap: add RabbitMQ, Redis Geo, Observability, K8s Helm charts, load tests, dashboard UI.

## Quick Start (Dev)
```bash
# 1) Replace the module path in go.mod with your GitHub path (optional but recommended):
#    module github.com/yourname/intercity-sim  →  module github.com/<you>/intercity-sim
make run

# In another terminal, create a trip via the gateway:
curl -X POST http://localhost:8080/api/v1/trips \
  -H "Content-Type: application/json" \
  -d '{
    "passenger_id":"p-1",
    "origin":{"lat":35.6892, "lng":51.3890, "city":"TEH"},
    "destination":{"lat":36.2605, "lng":59.6168, "city":"MHD"},
    "vehicle_type":"sedan"
  }' | jq

# Find the created trip by id (replace <id>):
curl http://localhost:8081/trips/<id> | jq
```

## Services
- **api-gateway**: `:8080`  
  - `POST /api/v1/trips` → returns created trip (with quoted_price)
  - `GET /health`
- **pricing-service**: `:8082`
  - `POST /price` → returns `{distance_km, base, per_km, surge, final}`
  - `GET /health`
- **trip-service**: `:8081`
  - `POST /trips` → create trip (persisted in Postgres)
  - `GET /trips/{id}` → fetch by id
  - `GET /health`

## CI
- GitHub Actions workflow runs `go test` on every push/PR.
- On `main`, it builds & pushes Docker images to **GHCR** (enable "Read and write" permissions for Actions in repo settings → Packages).

## Structure
```
/services/
  api-gateway/
  pricing-service/
  trip-service/
docker-compose.dev.yml
.github/workflows/ci.yml
go.mod
Makefile
```

## Next
- Add RabbitMQ + event-driven flow (`trip.requested` → `trip.priced` → `trip.matched`).
- Introduce Redis Geo for nearest-driver lookup.
- Add Prometheus/Grafana and Jaeger (OpenTelemetry).
- Helm charts + K8s deploy.
```
