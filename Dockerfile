FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o edgecom ./cmd/main.go

FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/edgecom /app/edgecom
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/edgecom"]
