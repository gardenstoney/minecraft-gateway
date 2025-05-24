# Build stage
FROM golang:1.24.1 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /src/out/gateway /src/cmd/gateway

# Runtime stage (distroless image)
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /src/out/gateway /
USER nonroot
WORKDIR /
CMD ["/gateway"]