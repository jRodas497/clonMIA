# Backend Dockerfile (multi-stage)
# Build the Go binary
FROM golang:1.23.1-alpine AS builder

ENV CGO_ENABLED=0
WORKDIR /src

# Copy go.mod/go.sum first to cache modules
COPY ./BackEnd/go.mod ./BackEnd/go.sum ./BackEnd/
WORKDIR /src/BackEnd
RUN go mod download

# Copy the rest of the backend sources
COPY ./BackEnd .

# Build the binary
RUN go build -ldflags="-s -w" -o backend .

# Final image: small Alpine with ca-certificates
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/BackEnd/backend ./backend

EXPOSE 3000
# Runtime (can be overridden with CMD in docker-compose or kubernetes)
CMD ["./backend"]
