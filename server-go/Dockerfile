FROM golang:1.23-alpine AS builder

# Build-time arguments
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG CONTAINER_BUILD_DATE=unknown

# Install build dependencies including libvips for image processing and HEIC support
RUN apk add --no-cache gcc g++ musl-dev git vips-dev pkgconfig

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

# Build with ldflags to embed version and build date
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-X 'main.Version=${VERSION}' -X 'main.BuildDate=${BUILD_DATE}' -X 'main.ContainerBuildDate=${CONTAINER_BUILD_DATE}'" \
    -o /photosync-server ./cmd/server

FROM alpine:latest

# Install runtime dependencies
# - ca-certificates: HTTPS connections
# - tzdata: timezone support
# - vips: image processing (thumbnails, format conversion)
# - libheif: HEIC/HEIF image support
# - exiftool: metadata embedding for photo IDs
RUN apk add --no-cache ca-certificates tzdata vips libheif exiftool

WORKDIR /app

COPY --from=builder /photosync-server /app/photosync-server
COPY --from=builder /app/web /app/web

# Create data directory for persistent storage
RUN mkdir -p /app/data

# Declare volumes for persistent data
# Users MUST mount this volume to persist data across container restarts/deployments
VOLUME ["/app/data"]

# Set environment variables to use the data directory
ENV DATABASE_PATH=/app/data/photosync.db
ENV PHOTO_STORAGE_PATH=/app/data/photos

EXPOSE 5000

CMD ["/app/photosync-server"]
