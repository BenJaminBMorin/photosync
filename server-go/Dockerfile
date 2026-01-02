FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=1 GOOS=linux go build -o /photosync-server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /photosync-server /app/photosync-server

EXPOSE 5000

CMD ["/app/photosync-server"]
