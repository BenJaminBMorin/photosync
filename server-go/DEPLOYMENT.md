# PhotoSync Deployment Guide

## ⚠️ CRITICAL: Data Persistence

**Your photos and database will be LOST if you don't properly configure persistent storage.**

PhotoSync stores data in `/app/data`:
- Database: `/app/data/photosync.db` (SQLite) or configured PostgreSQL
- Photos: `/app/data/photos/`

You **MUST** mount this directory as a volume to persist data across deployments.

## Quick Start with Docker Compose (Recommended)

The easiest way to deploy PhotoSync is with docker-compose:

```bash
# Start the server
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the server
docker-compose down

# Update to latest version (preserves data)
docker-compose pull
docker-compose up -d
```

Your data will be stored in `./data` on the host machine and will persist across updates.

## Manual Docker Run

If you prefer to run Docker manually:

```bash
# Create a directory for persistent data
mkdir -p /path/to/photosync-data

# Run the container with mounted volume
docker run -d \
  --name photosync \
  -p 5050:5000 \
  -v /path/to/photosync-data:/app/data \
  -e DATABASE_PATH=/app/data/photosync.db \
  -e PHOTO_STORAGE_PATH=/app/data/photos \
  --restart unless-stopped \
  photosync-server:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `/app/data/photosync.db` | Path to SQLite database file |
| `PHOTO_STORAGE_PATH` | `/app/data/photos` | Directory for photo storage |
| `DATABASE_URL` | (none) | PostgreSQL connection string (if using PostgreSQL instead of SQLite) |
| `PORT` | `5000` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## Using PostgreSQL (Production)

For production deployments, PostgreSQL is recommended over SQLite:

1. Uncomment the `postgres` service in `docker-compose.yml`
2. Set the `DATABASE_URL` environment variable:
   ```
   DATABASE_URL=postgres://photosync:yourpassword@postgres:5432/photosync
   ```
3. Comment out or remove the `DATABASE_PATH` variable

## Backup Your Data

### SQLite Backup

```bash
# Stop the server
docker-compose down

# Copy the data directory
cp -r ./data ./data-backup-$(date +%Y%m%d)

# Restart the server
docker-compose up -d
```

### PostgreSQL Backup

```bash
# Backup database
docker-compose exec postgres pg_dump -U photosync photosync > backup-$(date +%Y%m%d).sql

# Restore database
docker-compose exec -T postgres psql -U photosync photosync < backup.sql
```

## Deployment Best Practices

1. **Always use volumes**: Never run without mounting `/app/data`
2. **Regular backups**: Schedule automated backups of your data directory
3. **Test restores**: Periodically test your backup restoration process
4. **Use PostgreSQL for production**: SQLite is great for testing but PostgreSQL is better for production
5. **Monitor disk space**: Photos can consume significant storage
6. **Use HTTPS**: Put PhotoSync behind a reverse proxy (nginx, Caddy, Traefik) with HTTPS

## Troubleshooting Data Loss

If you've already lost data due to missing volumes:

1. **Check if old containers still exist**:
   ```bash
   docker ps -a
   ```

2. **If the old container exists, copy data out**:
   ```bash
   docker cp photosync:/app/data ./recovered-data
   ```

3. **Set up proper volumes and restart**:
   ```bash
   # Copy recovered data
   cp -r recovered-data ./data

   # Start with proper volumes
   docker-compose up -d
   ```

## Health Checks

PhotoSync includes a health endpoint at `/health`. Use it to monitor server status:

```bash
curl http://localhost:5050/health
```

## Updating PhotoSync

```bash
# Pull latest changes
git pull

# Rebuild and restart (preserves data in mounted volumes)
docker-compose up -d --build

# Or if using Docker Hub image
docker-compose pull
docker-compose up -d
```

Your data in the mounted volumes will be preserved during updates.

## Example Production Setup with Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name photos.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://localhost:5050;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/BenJaminBMorin/photosync/issues
- Check logs: `docker-compose logs -f photosync`
