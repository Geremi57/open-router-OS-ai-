FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ai-agent server.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ai-agent .
EXPOSE 8080
CMD ["./ai-agent"]