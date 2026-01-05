FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev git

# Install swag for swagger doc generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

# Copy go.mod first to cache dependencies
COPY go.mod go.sum* ./

# Copy source code
COPY . .

# Generate swagger docs first (creates docs package)
RUN swag init -g cmd/server/main.go -o docs

# Now run go mod tidy (docs package exists now)
RUN go mod tidy

RUN CGO_ENABLED=1 GOOS=linux go build -o /photosync-server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /photosync-server /app/photosync-server

EXPOSE 5000

CMD ["/app/photosync-server"]
