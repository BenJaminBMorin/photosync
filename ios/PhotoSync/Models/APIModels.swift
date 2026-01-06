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
