# Photo Sync Application - Implementation Plan

## Overview

A two-part photo synchronization system:
1. **Android App** - On-demand photo selection and upload
2. **.NET Backend Server** - Receives, tracks, and organizes photos into Year/Month folder structure

---

## System Architecture

```
┌─────────────────────┐         HTTPS/REST          ┌─────────────────────┐
│                     │ ──────────────────────────► │                     │
│   Android App       │                             │   .NET Backend      │
│                     │ ◄────────────────────────── │                     │
│  - Photo Gallery    │      JSON Responses         │  - REST API         │
│  - Sync Tracking    │                             │  - SQLite/SQL DB    │
│  - Multi-select     │                             │  - File Storage     │
│  - Upload Manager   │                             │  - Year/Month Org   │
└─────────────────────┘                             └─────────────────────┘
```

---

## Part 1: Android Application

### Technology Stack
| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Kotlin | Modern Android standard, null-safe, concise |
| Min SDK | 26 (Android 8.0) | Covers 95%+ of devices, modern APIs |
| UI | Jetpack Compose | Modern declarative UI, easier state management |
| HTTP Client | Retrofit + OkHttp | Industry standard, multipart upload support |
| Image Loading | Coil | Kotlin-first, efficient memory management |
| Local DB | Room | SQLite abstraction for sync tracking |
| DI | Hilt | Standard Android dependency injection |

### Core Features

#### 1. Photo Discovery
- Scan device for photos using MediaStore API
- Query: `MediaStore.Images.Media.EXTERNAL_CONTENT_URI`
- Retrieve: path, filename, date taken, size, file hash (MD5/SHA256)

#### 2. Sync Tracking (Local SQLite via Room)
```
Table: synced_photos
├── id (PRIMARY KEY)
├── device_path (TEXT, UNIQUE)
├── file_hash (TEXT) - MD5 or SHA256 for change detection
├── file_size (INTEGER)
├── date_taken (INTEGER) - epoch millis
├── synced_at (INTEGER) - epoch millis, NULL if never synced
├── server_photo_id (TEXT) - returned from server after upload
└── last_modified (INTEGER) - file modification timestamp
```

#### 3. Unsynced Photo Detection Logic
```
Photo is "unsynced" if:
  - NOT in synced_photos table

(Simple! Once synced, it stays synced. Edits don't trigger re-sync.)
```

#### 4. User Interface Screens

**Screen 1: Main/Gallery View**
```
┌────────────────────────────────────────┐
│  PhotoSync                    [Sync ▶] │
├────────────────────────────────────────┤
│  [Select All] [Clear]     12 selected  │
├────────────────────────────────────────┤
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐      │
│  │ ☑   │ │ ☐   │ │ ☑   │ │ ☐   │      │
│  │photo│ │photo│ │photo│ │photo│      │
│  └─────┘ └─────┘ └─────┘ └─────┘      │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐      │
│  │ ☑   │ │ ☐   │ │ ☑   │ │ ☐   │      │
│  │photo│ │photo│ │photo│ │photo│      │
│  └─────┘ └─────┘ └─────┘ └─────┘      │
│                                        │
│  Showing: [All ▼] [Unsynced Only ▼]   │
└────────────────────────────────────────┘
```

**Screen 2: Settings**
- Server URL configuration
- Authentication token/credentials
- Auto-scan on app open toggle
- Wi-Fi only sync toggle

**Screen 3: Sync Progress**
```
┌────────────────────────────────────────┐
│  Syncing Photos...                     │
├────────────────────────────────────────┤
│                                        │
│  ████████████░░░░░░░░  45/100         │
│                                        │
│  Current: IMG_20240315_142355.jpg      │
│  Speed: 2.3 MB/s                       │
│                                        │
│           [Cancel]                     │
└────────────────────────────────────────┘
```

#### 5. Upload Implementation
- Use multipart/form-data for file upload
- Include metadata: original filename, date taken, file hash
- Implement retry logic with exponential backoff
- Support background upload via WorkManager (optional enhancement)
- Chunk large files if needed (configurable threshold)

### Android Project Structure
```
app/
├── src/main/java/com/photosync/
│   ├── PhotoSyncApplication.kt
│   ├── di/
│   │   └── AppModule.kt (Hilt modules)
│   ├── data/
│   │   ├── local/
│   │   │   ├── PhotoDatabase.kt
│   │   │   ├── SyncedPhotoDao.kt
│   │   │   └── SyncedPhotoEntity.kt
│   │   ├── remote/
│   │   │   ├── PhotoSyncApi.kt (Retrofit interface)
│   │   │   └── ApiModels.kt
│   │   └── repository/
│   │       └── PhotoRepository.kt
│   ├── domain/
│   │   ├── model/
│   │   │   └── Photo.kt
│   │   └── usecase/
│   │       ├── GetUnsyncedPhotosUseCase.kt
│   │       ├── SyncPhotosUseCase.kt
│   │       └── ScanDevicePhotosUseCase.kt
│   └── ui/
│       ├── theme/
│       ├── gallery/
│       │   ├── GalleryScreen.kt
│       │   └── GalleryViewModel.kt
│       ├── settings/
│       │   ├── SettingsScreen.kt
│       │   └── SettingsViewModel.kt
│       └── sync/
│           ├── SyncProgressScreen.kt
│           └── SyncViewModel.kt
└── src/main/res/
    └── ... (resources)
```

---

## Part 2: .NET Backend Server

### Technology Stack
| Component | Technology | Rationale |
|-----------|------------|-----------|
| Framework | .NET 8 | Latest LTS, best performance |
| API | ASP.NET Core Minimal APIs | Lightweight, fast for simple CRUD |
| Database | SQLite (dev) / PostgreSQL (prod) | Easy setup, scalable option |
| ORM | Entity Framework Core | Standard .NET ORM |
| File Storage | Local filesystem | Simple, NAS-friendly |
| Auth | API Key or JWT | Simple security |

### Core Features

#### 1. Photo Storage Organization
```
/photos/
├── 2024/
│   ├── 01/
│   │   ├── IMG_20240115_093045_abc123.jpg
│   │   └── IMG_20240120_184532_def456.jpg
│   ├── 02/
│   │   └── ...
│   └── 12/
│       └── ...
├── 2025/
│   ├── 01/
│   └── ...
└── ...
```

**Filename format:** `{original_name}_{unique_suffix}.{ext}`
- Preserves original filename
- Adds unique suffix to prevent collisions
- Extension preserved

#### 2. Database Schema

```sql
-- Main photos table (simplified - no EXIF/metadata)
CREATE TABLE photos (
    id TEXT PRIMARY KEY,  -- GUID
    original_filename TEXT NOT NULL,
    stored_path TEXT NOT NULL,  -- e.g., "2024/03/IMG_xxx.jpg"
    file_hash TEXT NOT NULL,  -- SHA256 for duplicate detection
    file_size INTEGER NOT NULL,
    date_taken DATETIME NOT NULL,
    uploaded_at DATETIME NOT NULL
);

CREATE INDEX idx_photos_hash ON photos(file_hash);
CREATE INDEX idx_photos_date ON photos(date_taken);
```

#### 3. REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/photos/upload` | Upload single photo (multipart) |
| `POST` | `/api/photos/upload/batch` | Upload multiple photos |
| `GET` | `/api/photos/check` | Check if photos exist by hash |
| `GET` | `/api/photos` | List all photos (paginated) |
| `GET` | `/api/photos/{id}` | Get photo metadata |
| `DELETE` | `/api/photos/{id}` | Delete a photo |
| `GET` | `/api/health` | Health check endpoint |

#### 4. API Request/Response Models

**Upload Request (multipart/form-data):**
```
file: <binary>
originalFilename: "IMG_20240315_142355.jpg"
dateTaken: "2024-03-15T14:23:55Z"
fileHash: "sha256:abc123..."
deviceId: "device-uuid" (optional)
```

**Upload Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "storedPath": "2024/03/IMG_20240315_142355_abc123.jpg",
  "uploadedAt": "2024-03-15T18:30:00Z",
  "isDuplicate": false
}
```

**Check Hashes Request:**
```json
{
  "hashes": [
    "sha256:abc123...",
    "sha256:def456...",
    "sha256:ghi789..."
  ]
}
```

**Check Hashes Response:**
```json
{
  "existing": ["sha256:abc123..."],
  "missing": ["sha256:def456...", "sha256:ghi789..."]
}
```

#### 5. Duplicate Detection
- Before storing, check if file_hash exists in database
- If duplicate: return existing photo ID, set `isDuplicate: true`
- If new: store file, create record, return new ID

#### 6. Update Detection
- Client sends hash + server_photo_id for previously synced photos
- If hash changed: treat as new upload, optionally archive old version
- Track versions if needed (enhancement)

### .NET Project Structure
```
PhotoSyncServer/
├── PhotoSyncServer.sln
├── src/
│   └── PhotoSyncServer/
│       ├── Program.cs
│       ├── appsettings.json
│       ├── appsettings.Development.json
│       ├── Data/
│       │   ├── PhotoDbContext.cs
│       │   └── Migrations/
│       ├── Models/
│       │   ├── Photo.cs
│       │   ├── Device.cs
│       │   └── Dtos/
│       │       ├── UploadRequest.cs
│       │       ├── UploadResponse.cs
│       │       ├── CheckHashesRequest.cs
│       │       └── CheckHashesResponse.cs
│       ├── Services/
│       │   ├── IPhotoStorageService.cs
│       │   ├── PhotoStorageService.cs
│       │   ├── IHashService.cs
│       │   └── HashService.cs
│       ├── Endpoints/
│       │   └── PhotoEndpoints.cs
│       └── Middleware/
│           └── ApiKeyAuthMiddleware.cs
└── tests/
    └── PhotoSyncServer.Tests/
        └── ...
```

---

## Security Considerations

### Authentication Options (Choose One)

**Option A: API Key (Simpler)**
- Generate random API key, store in server config
- Client sends in header: `X-API-Key: your-secret-key`
- Good for personal/home use

**Option B: JWT Tokens (More Robust)**
- Login endpoint returns JWT
- Client sends: `Authorization: Bearer <token>`
- Better for multi-user scenarios

### Additional Security
- HTTPS required (self-signed cert OK for home NAS)
- Rate limiting on upload endpoints
- Max file size limits (configurable)
- Allowed file types validation (JPEG, PNG, HEIC, etc.)
- Sanitize filenames to prevent path traversal

---

## Implementation Phases

### Phase 1: Core Backend (Foundation)
1. Create .NET 8 Web API project
2. Set up Entity Framework Core with SQLite
3. Implement Photo entity and migrations
4. Create PhotoStorageService (save files to Year/Month folders)
5. Implement `/api/photos/upload` endpoint
6. Implement `/api/photos/check` endpoint
7. Add basic API key authentication
8. Add health check endpoint
9. Test with Postman/curl

### Phase 2: Core Android App
1. Create new Android project (Kotlin, Compose)
2. Set up Room database for sync tracking
3. Implement MediaStore photo scanning
4. Create photo gallery UI with grid view
5. Implement multi-select functionality
6. Add settings screen (server URL, API key)
7. Implement Retrofit client for API calls
8. Create basic upload functionality (single photo)

### Phase 3: Full Sync Flow
1. Implement hash computation on Android
2. Add "check hashes" call before upload (avoid duplicates)
3. Implement batch/sequential upload with progress
4. Update local database after successful uploads
5. Add "unsynced only" filter to gallery
6. Implement retry logic for failed uploads
7. Add sync progress UI with cancel option

### Phase 4: Polish & Enhancements
1. Add proper error handling and user feedback
2. Implement Wi-Fi only sync option
3. Add pull-to-refresh in gallery
4. Optimize thumbnail loading (caching)
5. Add basic logging on server
6. Write basic tests for critical paths
7. Documentation

---

## Configuration

### Server Configuration (appsettings.json)
```json
{
  "PhotoStorage": {
    "BasePath": "/mnt/nas/photos",
    "MaxFileSizeMB": 50,
    "AllowedExtensions": [".jpg", ".jpeg", ".png", ".heic", ".gif", ".webp"]
  },
  "Security": {
    "ApiKey": "your-secret-api-key-here"
  },
  "ConnectionStrings": {
    "PhotoDb": "Data Source=photosync.db"
  }
}
```

### Android Configuration (Stored in SharedPreferences/DataStore)
- Server URL: `https://your-nas-ip:5000`
- API Key: User-entered
- Wi-Fi Only: true/false
- Auto-scan: true/false

---

## Design Decisions (Confirmed)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Authentication | API Key | Simple, sufficient for personal NAS use |
| Edited Photos | Count as synced | Only care about initial backup, not versions |
| EXIF/Metadata | Skip | Keep it simple, just store the files |
| Thumbnails | Skip | Not needed, no web UI planned |
| Sync Tracking | By file path + initial sync | Once synced, stays synced unless deleted from tracking |

---

## Estimated Effort by Component

| Component | Complexity | Core Files |
|-----------|------------|------------|
| Backend API | Medium | ~10 files |
| Backend Storage Logic | Low | ~3 files |
| Android Photo Scanning | Medium | ~4 files |
| Android Local Database | Low | ~3 files |
| Android UI (Gallery) | Medium | ~4 files |
| Android Upload Logic | Medium | ~4 files |
| Android Settings | Low | ~2 files |

**Total estimated files:** ~30 files across both projects

---

## Next Steps

Once you approve this plan, I will:
1. Create the .NET backend project structure
2. Implement core upload and storage functionality
3. Create the Android project structure
4. Implement the photo scanning and gallery UI
5. Connect both with the sync flow

**Please review and let me know:**
- Any changes to the architecture?
- Which authentication method? (API Key recommended for simplicity)
- Any features to add or remove?
- Any of the questions above you'd like to answer now?
