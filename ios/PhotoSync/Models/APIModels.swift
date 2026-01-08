import Foundation

/// Response from uploading a photo
struct UploadResponse: Codable {
    let id: String
    let storedPath: String
    let uploadedAt: String
    let isDuplicate: Bool
}

/// Request to check which hashes exist
struct CheckHashesRequest: Codable {
    let hashes: [String]
}

/// Response from checking hashes
struct CheckHashesResponse: Codable {
    let existing: [String]
    let missing: [String]
}

/// Single photo in list response
struct PhotoResponse: Codable {
    let id: String
    let originalFilename: String
    let storedPath: String
    let fileSize: Int64
    let dateTaken: String
    let uploadedAt: String
    let hash: String?
    let thumbnailPath: String?
}

/// Response from listing photos
struct PhotoListResponse: Codable {
    let photos: [PhotoResponse]
    let totalCount: Int
    let skip: Int
    let take: Int
}

/// Health check response
struct HealthResponse: Codable {
    let status: String
    let timestamp: String
}

/// Error response from server
struct ErrorResponse: Codable {
    let error: String
}

// MARK: - Device Registration

/// Request to register a device for push notifications
struct RegisterDeviceRequest: Codable {
    let fcmToken: String
    let deviceName: String
    let platform: String
}

/// Response from device registration
struct DeviceResponse: Codable {
    let id: String
    let deviceName: String
    let platform: String
    let registeredAt: String
    let lastSeenAt: String
    let isActive: Bool
}

// MARK: - Auth Response

/// Request to respond to a web authentication request
struct AuthResponseRequest: Codable {
    let requestId: String
    let approved: Bool
    let deviceId: String?
}

// MARK: - Delete Response

/// Request to respond to a photo deletion request
struct DeleteResponseRequest: Codable {
    let requestId: String
    let approved: Bool
    let deviceId: String?
}

// MARK: - Invite Redemption

/// Request to redeem an invite token
struct RedeemInviteRequest: Codable {
    let token: String
    let deviceInfo: String?
}

/// Response from redeeming an invite token
struct RedeemInviteResponse: Codable {
    let serverUrl: String
    let apiKey: String
    let email: String
    let userId: String
}

/// Sync progress tracking
struct SyncProgress {
    let total: Int
    var completed: Int
    var failed: Int
    var currentFileName: String?
    var isCancelled: Bool = false
    var sequence: Int = 0  // Sequence number to ensure ordered updates

    var progressPercent: Double {
        total > 0 ? Double(completed) / Double(total) : 0
    }

    var isComplete: Bool {
        completed + failed >= total
    }
}

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
