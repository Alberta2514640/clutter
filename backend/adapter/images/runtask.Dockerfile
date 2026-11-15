# B's run task (backend/adapter/cmd/runtask): one short-lived Fargate task per run.
# Built from the backend/ folder:
#   docker build -f backend/adapter/images/runtask.Dockerfile backend
# backend/adapter/images/build-and-push.sh builds and pushes it with a version tag.
FROM golang:1.25.3 AS build
WORKDIR /src
COPY api/go.mod api/go.sum api/
COPY adapter/go.mod adapter/go.sum adapter/
RUN go -C adapter mod download
COPY api/ api/
COPY adapter/ adapter/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C adapter build -trimpath -o /out/runtask ./cmd/runtask

# The same Terraform version as A's deployer image
# (backend/infra/modules/fargate/ecr-scripts/terraform-deployer/Dockerfile).
FROM hashicorp/terraform:1.14.4 AS terraform

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
 && adduser -D -H -u 10001 runner \
 && mkdir /work \
 && chown runner:runner /work
COPY --from=terraform /bin/terraform /usr/local/bin/terraform
COPY --from=build /out/runtask /usr/local/bin/runtask
USER 10001:10001
WORKDIR /work
ENTRYPOINT ["/usr/local/bin/runtask"]
