FROM golang:1.25.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /payment ./cmd/server
FROM alpine:3.19

WORKDIR /

COPY --from=builder /payment /payment
COPY .env .env

EXPOSE 8080

CMD ["/payment"]