# ⚠️ CRITICAL: Data Persistence Fix

## What Changed

PhotoSync now properly supports data persistence across deployments. **Previous deployments may have lost data** because volumes were not properly configured.

## The Problem

Before this fix:
- Database and photos were stored inside the Docker container
- When deploying a new version, the old container was destroyed
- **All your photos and database were deleted with the container**

## The Solution

The Dockerfile now:
1. Declares `/app/data` as a VOLUME
2. Sets default paths to use `/app/data`:
   - Database: `/app/data/photosync.db`
   - Photos: `/app/data/photos/`

## What You Need To Do

### If Using Docker Compose (Recommended)

Use the provided `docker-compose.yml`:

```bash
docker-compose up -d
```

This automatically mounts `./data` to `/app/data` in the container.

### If Using Docker Run

Add the volume mount:

```bash
docker run -d \
  --name photosync \
  -p 5050:5000 \
  -v /path/to/data:/app/data \
  photosync-server:latest
```

### If Already Lost Data

If you just deployed and lost data, check if the old container still exists:

```bash
# List all containers (including stopped ones)
docker ps -a

# If the old container exists, extract the data
docker cp OLD_CONTAINER_NAME:/app/data ./recovered-data

# Then use docker-compose or docker run with proper volumes
```

## Verification

After deploying with proper volumes, verify data persists:

```bash
# Upload a test photo
# Deploy a new version
docker-compose up -d --build

# Check if photo still exists - it should!
```

## Documentation

See [DEPLOYMENT.md](./DEPLOYMENT.md) for complete deployment instructions.

## Questions?

Open an issue on GitHub if you need help recovering data or setting up persistent volumes.
