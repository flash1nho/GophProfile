FROM golang:1.25.1-alpine AS builder

RUN apk add --no-cache \
    build-base \
    gcc \
    musl-dev \
    libwebp-dev \
    pkgconfig

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    libwebp

WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/worker .
COPY --from=builder /app/web ./web/

CMD ["./server"]
