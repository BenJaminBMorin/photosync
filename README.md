# PhotoSync

A photo synchronization system for syncing photos from Android devices to a NAS server.

![CI](../../actions/workflows/ci.yml/badge.svg)

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

### Server (Go)

Alternative implementation in Go with identical API signatures.

**Features:**
- Same API endpoints as .NET version
- Same API Key authentication
- Same Year/Month folder organization
- SQLite database
- Smaller binary, lower memory footprint

**Location:** `server-go/`

**Run:**
```bash
cd server-go
go run ./cmd/server
```

**Configuration:** Copy `config.example.json` to `config.json` and edit:
```json
{
  "serverAddress": ":5000",
  "photoStorage": {
    "basePath": "/path/to/photos",
    "maxFileSizeMB": 50
  },
  "security": {
    "apiKey": "your-secure-api-key-32-chars-minimum"
  }
}
```

Or use environment variables:
```bash
export API_KEY="your-secure-api-key"
export PHOTO_STORAGE_PATH="/path/to/photos"
export SERVER_ADDRESS=":5000"
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

Both servers implement identical endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/photos/upload` | Upload a photo (multipart/form-data) |
| POST | `/api/photos/check` | Check if hashes exist |
| GET | `/api/photos` | List photos (paginated) |
| GET | `/api/photos/{id}` | Get photo by ID |
| DELETE | `/api/photos/{id}` | Delete photo |
| GET | `/api/health` | Health check |

### Request/Response Examples

**Upload Photo:**
```bash
curl -X POST http://localhost:5000/api/photos/upload \
  -H "X-API-Key: your-api-key" \
  -F "file=@photo.jpg" \
  -F "originalFilename=photo.jpg" \
  -F "dateTaken=2024-03-15T14:30:00Z"
```

**Check Hashes:**
```bash
curl -X POST http://localhost:5000/api/photos/check \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"hashes": ["abc123...", "def456..."]}'
```

## Usage

1. Start the server on your NAS (.NET or Go version)
2. Configure the API key
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
├── server-go/                 # Go Backend
│   ├── cmd/server/            # Entry point
│   └── internal/
│       ├── config/            # Configuration
│       ├── handlers/          # HTTP handlers
│       ├── middleware/        # Auth middleware
│       ├── models/            # Domain models
│       ├── repository/        # Database layer
│       └── services/          # Business logic
├── android/                   # Android App
│   └── app/src/
│       ├── main/java/com/photosync/
│       │   ├── data/          # Repositories, API
│       │   ├── domain/        # Models
│       │   ├── di/            # Dependency injection
│       │   └── ui/            # Compose screens
│       └── test/              # Unit tests
├── .github/workflows/         # CI/CD
│   ├── ci.yml                 # Combined CI
│   ├── server-dotnet.yml      # .NET build + publish
│   ├── server-go.yml          # Go build + publish
│   └── android.yml            # Android build
└── PLAN.md                    # Implementation plan
```

## CI/CD

GitHub Actions automatically:
- **On PR/Push:** Builds and tests all components
- **On Push to main:** Creates release artifacts:
  - .NET: Self-contained binaries for linux-x64, linux-arm64, win-x64, osx-x64, osx-arm64
  - Go: Static binaries for linux-amd64, linux-arm64, windows-amd64, darwin-amd64, darwin-arm64
  - Android: Debug and Release APKs

## Choosing Between .NET and Go Servers

| Aspect | .NET | Go |
|--------|------|-----|
| Binary Size | ~30-50MB (self-contained) | ~10-15MB |
| Memory Usage | Higher | Lower |
| Startup Time | Slower | Faster |
| Dependencies | .NET Runtime (if not self-contained) | None (static binary) |
| Performance | Excellent | Excellent |
| Ecosystem | Rich, Entity Framework | Minimal dependencies |

Both servers are functionally identical and can be used interchangeably.

## Security

- API Key authentication for all API endpoints (except health check)
- Constant-time comparison to prevent timing attacks
- Path traversal protection in file storage
- File extension validation
- Request size limits

## License

MIT
