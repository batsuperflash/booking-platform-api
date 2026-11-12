FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o booking-platform-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/booking-platform-api .

EXPOSE 8080

CMD ["./booking-platform-api"]