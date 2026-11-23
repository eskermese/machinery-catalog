# Step 1: Prepare
FROM golang:1.26-alpine3.22 AS builder

RUN apk add --no-cache unzip curl make bash

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=vendor -o /bin/app ./cmd/app/

# Step 2: Final
FROM alpine:3.22

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/version /app/version
COPY --from=builder /app/swagger /app/swagger
COPY --from=builder /app/config /app/config

COPY --from=builder /bin/app /app/main
RUN chmod +x /app/main

ENTRYPOINT ["/app/main"]
