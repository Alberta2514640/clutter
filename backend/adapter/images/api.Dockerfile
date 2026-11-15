# B API (backend/adapter/cmd/api), the service on Fargate behind B's load balancer.
# Built from the backend/ folder, because the adapter imports backend/api/generic:
#   docker build -f backend/adapter/images/api.Dockerfile backend
# backend/adapter/images/build-and-push.sh builds and pushes it with a version tag.
FROM golang:1.25.3 AS build
WORKDIR /src
COPY api/go.mod api/go.sum api/
COPY adapter/go.mod adapter/go.sum adapter/
RUN go -C adapter mod download
COPY api/ api/
COPY adapter/ adapter/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C adapter build -trimpath -o /out/adapter-api ./cmd/api

# No shell and no Terraform: runs happen in the run task (runtask.Dockerfile).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/adapter-api /app/adapter-api
COPY storage-sql/adapter-tables.sql /app/adapter-tables.sql
ENV ADAPTER_LISTEN_ADDR=0.0.0.0:8080 \
    ADAPTER_SCHEMA_FILE=/app/adapter-tables.sql \
    ADAPTER_DATA_DIR=/tmp/adapter
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/adapter-api"]
