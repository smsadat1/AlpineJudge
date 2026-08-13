# Build the binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY dispatcher/ ./dispatcher
COPY shared/ ./shared
RUN go work init ./dispatcher ./shared
COPY dispatcher/go.mod dispatcher/go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ajdispatcher ./dispatcher/cmd

# Run as daemon 
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ajdispatcher .
CMD ["./ajdispatcher"]
