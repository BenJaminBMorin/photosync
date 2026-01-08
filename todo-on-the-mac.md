# PhotoSync iOS Updates - Smart Resync Feature

## Summary of Backend Changes (Completed)

The server-side "Smart Resync" feature has been fully implemented and tested. This document describes what was done and what iOS changes are needed to take advantage of it.

### What Was Implemented on the Server

1. **Database Schema Changes**
   - Added `origin_device_id` column to `photos` table (tracks which device uploaded each photo)
   - Created `device_sync_state` table (tracks sync progress per device)

2. **New API Endpoints**
   - `GET /api/sync/status` - Returns sync status with photo counts
   - `POST /api/sync/photos` - Bulk photo metadata with cursor-based pagination
   - `GET /api/sync/legacy-photos` - List photos without origin device
   - `POST /api/sync/claim-legacy` - Claim ownership of legacy photos
   - `GET /api/photos/{id}/download` - Full-res download with hash verification headers

3. **Upload Enhancement**
   - Upload endpoint now accepts optional `deviceId` form field
   - When provided, sets `origin_device_id` on the photo record

4. **Files Created/Modified**
   - `internal/models/sync_dto.go` - New sync request/response DTOs
   - `internal/models/device_sync_state.go` - New model
   - `internal/repository/device_sync_state_repository.go` - New repository
   - `internal/repository/interfaces.go` - Added sync methods to PhotoRepo
   - `internal/repository/photo_repository_postgres.go` - Implemented sync queries
   - `internal/repository/photo_repository_additions.go` - SQLite stubs
   - `internal/handlers/sync_handler.go` - New handler with all endpoints
   - `internal/handlers/photo_handler.go` - Added deviceId parameter
   - `cmd/server/main.go` - Registered sync routes

---

## iOS Changes Required

### Priority 1: Essential Device Tracking (Do This First)

These changes ensure photos uploaded from iOS have origin device tracking.

#### 1. Update `APIService.swift` - Add deviceId to upload

**File:** `ios/PhotoSync/Services/APIService.swift`

Change the `uploadPhoto` function signature (around line 34):

```swift
// FROM:
func uploadPhoto(
    imageData: Data,
    filename: String,
    dateTaken: Date
) async throws -> UploadResponse

// TO:
func uploadPhoto(
    imageData: Data,
    filename: String,
    dateTaken: Date,
    deviceId: String? = nil
) async throws -> UploadResponse
```

Then add the deviceId to the multipart form body (around line 60, after the dateTaken field):

```swift
// Add deviceId if provided
if let deviceId = deviceId {
    body.append("--\(boundary)\r\n".data(using: .utf8)!)
    body.append("Content-Disposition: form-data; name=\"deviceId\"\r\n\r\n".data(using: .utf8)!)
    body.append("\(deviceId)\r\n".data(using: .utf8)!)
}
```

#### 2. Update `SyncService.swift` - Pass deviceId when uploading

**File:** `ios/PhotoSync/Services/SyncService.swift`

Update the `syncPhotos` function to pass the device ID (around line 39):

```swift
// Get device ID from settings
let deviceId = AppSettings.deviceId

// Change the upload call from:
let response = try await api.uploadPhoto(
    imageData: imageData,
    filename: filename,
    dateTaken: photo.creationDate
)

// TO:
let response = try await api.uploadPhoto(
    imageData: imageData,
    filename: filename,
    dateTaken: photo.creationDate,
    deviceId: deviceId
)
```

---

### Priority 2: New Sync Endpoints (Enhanced Resync)

#### 3. Add New API Models

**File:** `ios/PhotoSync/Models/APIModels.swift`

Add these new model structs:

```swift
// MARK: - Sync Models

struct SyncStatusResponse: Codable {
    let totalPhotos: Int
    let devicePhotos: Int
    let otherDevicePhotos: Int
    let legacyPhotos: Int
    let lastSyncAt: Date?
    let serverVersion: Int
    let needsLegacyClaim: Bool
}

struct SyncPhotosRequest: Codable {
    let deviceId: String
    let cursor: String?
    let limit: Int
    let includeThumbnailUrls: Bool
    let sinceTimestamp: Date?
}

struct SyncPhotosResponse: Codable {
    let photos: [SyncPhotoItem]
    let pagination: PaginationInfo
    let sync: SyncInfo
}

struct SyncPhotoItem: Codable {
    let id: String
    let fileHash: String
    let originalFilename: String
    let fileSize: Int64
    let dateTaken: Date
    let uploadedAt: Date
    let originDevice: OriginDeviceInfo?
    let thumbnailUrl: String?
    let width: Int?
    let height: Int?
}

struct OriginDeviceInfo: Codable {
    let id: String
    let name: String
    let platform: String
    let isCurrentDevice: Bool
}

struct PaginationInfo: Codable {
    let cursor: String?
    let hasMore: Bool
}

struct SyncInfo: Codable {
    let totalCount: Int
    let returnedCount: Int
    let serverVersion: Int
}

struct LegacyPhotosResponse: Codable {
    let photos: [SyncPhotoItem]
    let totalCount: Int
    let message: String
}

struct ClaimLegacyRequest: Codable {
    let deviceId: String
    let claimAll: Bool
    let photoIds: [String]?
}

struct ClaimLegacyResponse: Codable {
    let claimed: Int
    let alreadyClaimed: Int
    let failed: Int
}
```

#### 4. Add New API Methods

**File:** `ios/PhotoSync/Services/APIService.swift`

Add these new methods:

```swift
// MARK: - Sync Endpoints

func getSyncStatus(deviceId: String?) async throws -> SyncStatusResponse {
    var request = URLRequest(url: baseURL.appendingPathComponent("api/sync/status"))
    request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
    if let deviceId = deviceId {
        request.setValue(deviceId, forHTTPHeaderField: "X-Device-ID")
    }

    let (data, response) = try await session.data(for: request)
    guard let httpResponse = response as? HTTPURLResponse,
          httpResponse.statusCode == 200 else {
        throw APIError.requestFailed
    }

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    return try decoder.decode(SyncStatusResponse.self, from: data)
}

func syncPhotos(request: SyncPhotosRequest) async throws -> SyncPhotosResponse {
    var urlRequest = URLRequest(url: baseURL.appendingPathComponent("api/sync/photos"))
    urlRequest.httpMethod = "POST"
    urlRequest.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
    urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601
    urlRequest.httpBody = try encoder.encode(request)

    let (data, response) = try await session.data(for: urlRequest)
    guard let httpResponse = response as? HTTPURLResponse,
          httpResponse.statusCode == 200 else {
        throw APIError.requestFailed
    }

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    return try decoder.decode(SyncPhotosResponse.self, from: data)
}

func getLegacyPhotos(limit: Int = 100) async throws -> LegacyPhotosResponse {
    var components = URLComponents(url: baseURL.appendingPathComponent("api/sync/legacy-photos"), resolvingAgainstBaseURL: false)!
    components.queryItems = [URLQueryItem(name: "limit", value: String(limit))]

    var request = URLRequest(url: components.url!)
    request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")

    let (data, response) = try await session.data(for: request)
    guard let httpResponse = response as? HTTPURLResponse,
          httpResponse.statusCode == 200 else {
        throw APIError.requestFailed
    }

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    return try decoder.decode(LegacyPhotosResponse.self, from: data)
}

func claimLegacyPhotos(deviceId: String, claimAll: Bool = true, photoIds: [String]? = nil) async throws -> ClaimLegacyResponse {
    var urlRequest = URLRequest(url: baseURL.appendingPathComponent("api/sync/claim-legacy"))
    urlRequest.httpMethod = "POST"
    urlRequest.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
    urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

    let request = ClaimLegacyRequest(deviceId: deviceId, claimAll: claimAll, photoIds: photoIds)
    urlRequest.httpBody = try JSONEncoder().encode(request)

    let (data, response) = try await session.data(for: urlRequest)
    guard let httpResponse = response as? HTTPURLResponse,
          httpResponse.statusCode == 200 else {
        throw APIError.requestFailed
    }

    return try JSONDecoder().decode(ClaimLegacyResponse.self, from: data)
}
```

#### 5. Update `AutoSyncManager.swift` - Use New Sync Endpoints

**File:** `ios/PhotoSync/Services/AutoSyncManager.swift`

Update the `resyncFromServer()` method to use cursor-based pagination instead of skip/take. The key change is replacing the `listPhotos()` calls with the new `syncPhotos()` method.

---

### Priority 3: UI Enhancements (Optional)

1. **Show origin device info** - Display which device uploaded each photo in ServerPhotosView
2. **Legacy claiming UI** - Add a prompt when `needsLegacyClaim` is true to let user claim existing photos
3. **Sync status display** - Show sync progress with device breakdown

---

## Testing the iOS Changes

After making the Priority 1 changes:

1. Build and run the app
2. Upload a new photo
3. Check the server logs or database to verify `origin_device_id` is set:
   ```sql
   SELECT id, original_filename, origin_device_id FROM photos ORDER BY uploaded_at DESC LIMIT 5;
   ```

After making Priority 2 changes:

1. Test `getSyncStatus()` - should return photo counts
2. Test `syncPhotos()` - should return paginated photos with cursor
3. Test the full resync flow with cursor pagination

---

## API Reference

### GET /api/sync/status
**Headers:** `X-API-Key`, `X-Device-ID` (optional)

**Response:**
```json
{
    "totalPhotos": 1277,
    "devicePhotos": 1277,
    "otherDevicePhotos": 0,
    "legacyPhotos": 0,
    "serverVersion": 1277,
    "needsLegacyClaim": false
}
```

### POST /api/sync/photos
**Headers:** `X-API-Key`, `Content-Type: application/json`

**Request:**
```json
{
    "deviceId": "uuid",
    "cursor": "optional-cursor",
    "limit": 100,
    "includeThumbnailUrls": true,
    "sinceTimestamp": "2024-01-01T00:00:00Z"
}
```

**Response:**
```json
{
    "photos": [{
        "id": "photo-uuid",
        "fileHash": "sha256",
        "originalFilename": "IMG_1234.jpg",
        "fileSize": 2048576,
        "dateTaken": "2024-01-15T10:30:00Z",
        "uploadedAt": "2024-01-15T10:35:00Z",
        "originDevice": {
            "id": "device-uuid",
            "name": "John's iPhone",
            "platform": "ios",
            "isCurrentDevice": true
        },
        "thumbnailUrl": "/api/web/photos/{id}/thumbnail",
        "width": 4032,
        "height": 3024
    }],
    "pagination": { "cursor": "next-cursor", "hasMore": true },
    "sync": { "totalCount": 1348, "returnedCount": 100, "serverVersion": 42 }
}
```

### POST /api/sync/claim-legacy
**Headers:** `X-API-Key`, `Content-Type: application/json`

**Request:**
```json
{
    "deviceId": "uuid",
    "claimAll": true
}
```

**Response:**
```json
{
    "claimed": 1277,
    "alreadyClaimed": 0,
    "failed": 0
}
```

### GET /api/photos/{id}/download
**Headers:** `X-API-Key`

**Response Headers:**
- `Content-Disposition: attachment; filename="IMG_1234.jpg"`
- `X-PhotoSync-Hash: sha256` (for verification)
- `X-PhotoSync-DateTaken: 2024-01-15T10:30:00Z`
- `Accept-Ranges: bytes` (for resumable downloads)
