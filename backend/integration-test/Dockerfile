FROM golang:1.26-alpine3.22 AS builder

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -tags=integration -mod=vendor -c -o /integration-test-bin ./integration-test

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /integration-test-bin ./integration-test
COPY --from=builder /app/config /config
COPY --from=builder /app/migrations /migrations

CMD ["./integration-test"]
