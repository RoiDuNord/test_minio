FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache mailcap

COPY --from=builder /app/main /s3_rest/main
COPY --from=builder /app/.env /s3_rest/.env

WORKDIR /s3_rest

EXPOSE 8080

CMD ["/s3_rest/main"]
