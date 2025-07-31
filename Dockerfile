FROM golang:1.19 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rbac-project ./cmd/app

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/rbac-project .
EXPOSE 8080
CMD ["./rbac-project"]