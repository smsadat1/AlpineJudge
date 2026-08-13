# Build the binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY runner/ ./runner
COPY shared/ ./shared
RUN go work init ./runner ./runner/utils ./shared
COPY runner/go.mod runner/go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ajrunner ./runner/cmd

# Run as daemon 
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ajrunner .
CMD ["./ajrunner"]
