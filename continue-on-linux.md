# Continue on Linux Server - PhotoSync Debugging

## Current Status

The iOS app was successfully built and deployed to TestFlight (build 159). However, the app appears to be stuck in a loading/spinning state when trying to fetch photos from the server.

## Problem to Investigate

The iOS app is not loading photos/images from the server. Possible causes:
1. API endpoint issues
2. Database query problems
3. Image/thumbnail serving issues
4. Network connectivity between app and server

## Key Files to Check

### Server (Node.js/Express)
- `server/src/routes/photos.ts` - Photo API endpoints
- `server/src/routes/sync.ts` - New sync endpoints (recently added)
- `server/src/index.ts` - Main server entry point

### iOS App (for reference)
- `ios/PhotoSync/Services/APIService.swift` - API calls
- `ios/PhotoSync/Services/AutoSyncManager.swift` - Sync logic

## Commands to Debug

### Check Server Status
```bash
docker ps | grep photosync
docker logs photosync-server --tail 100
docker logs photosync-server -f  # Follow logs in real-time
```

### Check if API is Responding
```bash
# Test health endpoint (adjust port/URL as needed)
curl http://localhost:3000/api/health

# Test photos endpoint (requires API key)
curl -H "X-API-Key: YOUR_API_KEY" http://localhost:3000/api/photos
```

### Check Database
```bash
docker exec -it photosync-db psql -U postgres -d photosync
# Then: SELECT COUNT(*) FROM photos;
# And: SELECT * FROM photos LIMIT 5;
```

## Suggestion: Add Proper Logging

Currently debugging requires SSH access to the server. Consider adding a logging service for remote debugging:

### Option 1: Logtail/Better Stack (recommended for small projects)
- Free tier available
- Easy to set up with Node.js
- Real-time log viewing

### Option 2: Papertrail
- Simple syslog-based logging
- Free tier for small volumes

### Option 3: Self-hosted Loki + Grafana
- More complex but full control
- Good if you want dashboards

### Quick Implementation (Logtail example)
```bash
npm install @logtail/node @logtail/winston
```

```typescript
// server/src/utils/logger.ts
import { Logtail } from '@logtail/node';
import winston from 'winston';
import { LogtailTransport } from '@logtail/winston';

const logtail = new Logtail(process.env.LOGTAIL_SOURCE_TOKEN || '');

export const logger = winston.createLogger({
  level: 'info',
  format: winston.format.json(),
  transports: [
    new winston.transports.Console(),
    new LogtailTransport(logtail),
  ],
});
```

Then replace `console.log` calls with `logger.info()`, `logger.error()`, etc.

## Recent Changes Context

### TestFlight Build Fixes (just completed)
1. Fixed fastlane match to use `readonly: true` for CI
2. Set default keychain for self-hosted runner
3. Added certificate cleanup to remove duplicates
4. Fixed Swift async/await errors in `BackgroundTaskManager.swift`

### Smart Resync Implementation (in progress)
According to the plan at `~/.claude/plans/iridescent-wiggling-narwhal.md`:
- Backend sync endpoints are deployed (`/api/sync/status`, `/api/sync/photos`, etc.)
- iOS needs to use these new endpoints for cursor-based pagination
- Device tracking is partially implemented

## Environment Info

- Server runs in Docker on Linux
- Database: PostgreSQL in Docker
- iOS app: Xcode 26.1.1, Swift
- Backend: Node.js/Express with TypeScript

## Next Steps

1. Check server logs for errors when the iOS app makes requests
2. Verify the API endpoints are responding correctly
3. Consider adding remote logging for easier debugging
4. If API is working, check if the issue is iOS-side (network, parsing, etc.)
