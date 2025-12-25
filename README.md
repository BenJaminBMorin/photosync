# PhotoSync

A photo synchronization system for syncing photos from Android devices to a NAS server.

## Components

### Server (.NET 8)

A REST API server that receives photos and organizes them into Year/Month folder structure.

**Features:**
- API Key authentication
- SHA256 hash-based duplicate detection
- Year/Month folder organization
- SQLite database for photo tracking

**Location:** `server/`

**Run:**
```bash
cd server/src/PhotoSync.Api
dotnet run
```

**Configuration:** Edit `appsettings.json`:
```json
{
  "PhotoStorage": {
    "BasePath": "/path/to/photos",
    "MaxFileSizeMB": 50
  },
  "Security": {
    "ApiKey": "your-secure-api-key-32-chars-minimum"
  }
}
```

### Android App (Kotlin + Jetpack Compose)

A mobile app for selecting and syncing photos.

**Features:**
- Photo gallery with grid view
- Multi-select support (individual or select all)
- Filter to show only unsynced photos
- Sync progress tracking
- Room database for tracking synced photos
- Server connection testing

**Location:** `android/`

**Build:**
```bash
cd android
./gradlew assembleDebug
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/photos/upload` | Upload a photo |
| POST | `/api/photos/check` | Check if hashes exist |
| GET | `/api/photos` | List photos (paginated) |
| GET | `/api/photos/{id}` | Get photo by ID |
| DELETE | `/api/photos/{id}` | Delete photo |
| GET | `/api/health` | Health check |

## Usage

1. Start the server on your NAS
2. Configure the API key in `appsettings.json`
3. Install the Android app
4. Open Settings and enter your server URL and API key
5. Test the connection
6. Select photos and tap Sync

## Project Structure

```
├── server/                    # .NET Backend
│   ├── src/
│   │   ├── PhotoSync.Api/     # Web API
│   │   ├── PhotoSync.Core/    # Domain layer
│   │   └── PhotoSync.Infrastructure/  # Data layer
│   └── tests/
│       └── PhotoSync.Tests/   # Unit tests
├── android/                   # Android App
│   └── app/src/
│       ├── main/java/com/photosync/
│       │   ├── data/          # Repositories, API
│       │   ├── domain/        # Models
│       │   ├── di/            # Dependency injection
│       │   └── ui/            # Compose screens
│       └── test/              # Unit tests
└── PLAN.md                    # Implementation plan
```

## Security

- API Key authentication for all API endpoints (except health check)
- Constant-time comparison to prevent timing attacks
- Path traversal protection in file storage
- File extension validation
- Request size limits

## License

MIT
