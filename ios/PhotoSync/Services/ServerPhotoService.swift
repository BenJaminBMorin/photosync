import Foundation
import Photos
import UIKit

/// Service for managing server-only photos (photos on server but not on device)
actor ServerPhotoService {
    static let shared = ServerPhotoService()

    private let api = APIService.shared
    private let photoLibrary = PhotoLibraryService.shared

    private init() {}

    /// Get paginated photos that exist on server but not from this device
    /// This uses the Smart Resync API which efficiently filters on the server side
    func getServerOnlyPhotos(cursor: String? = nil, limit: Int = 50) async throws -> ServerPhotoPage {
        await Logger.shared.info("Fetching server-only photos (cursor: \(cursor ?? "nil"), limit: \(limit))")

        guard let deviceId = AppSettings.deviceId else {
            throw ServerPhotoError.noDeviceId
        }

        // Use the syncPhotos endpoint to get photos with device information
        let request = SyncPhotosRequest(
            deviceId: deviceId,
            cursor: cursor,
            limit: limit,
            includeThumbnailUrls: true,  // Request thumbnail URLs from server
            sinceTimestamp: nil
        )

        let response = try await api.syncPhotos(request: request)

        // Filter to only photos NOT from this device (server-only photos)
        let serverOnlyPhotos = response.photos
            .filter { photo in
                // Include if no origin device (legacy) or if origin device is different
                if let originDevice = photo.originDevice {
                    return !originDevice.isCurrentDevice
                } else {
                    return true  // Include legacy photos without device info
                }
            }
            .map { ServerPhoto(from: $0) }

        await Logger.shared.info("Found \(serverOnlyPhotos.count) server-only photos on this page")

        return ServerPhotoPage(
            photos: serverOnlyPhotos,
            cursor: response.pagination.cursor,
            hasMore: response.pagination.hasMore
        )
    }

    /// Download thumbnail for a server photo
    func downloadThumbnail(for photo: ServerPhoto) async throws -> UIImage {
        let imageData = try await api.downloadThumbnail(photoId: photo.id)

        guard let image = UIImage(data: imageData) else {
            throw ServerPhotoError.invalidImageData
        }

        return image
    }

    /// Restore a photo from server to device photo library
    func restorePhoto(_ photo: ServerPhoto) async throws {
        await Logger.shared.info("Restoring photo \(photo.id) to device")

        // Download full photo
        let imageData = try await api.downloadPhoto(photoId: photo.id)

        guard let image = UIImage(data: imageData) else {
            throw ServerPhotoError.invalidImageData
        }

        // Save to photo library
        try await photoLibrary.saveImage(image, filename: photo.originalFilename)

        await Logger.shared.info("Successfully restored photo \(photo.id)")
    }
}

/// Paginated response for server-only photos
struct ServerPhotoPage {
    let photos: [ServerPhoto]
    let cursor: String?
    let hasMore: Bool
}

/// Represents a photo that exists on the server
struct ServerPhoto: Identifiable, Codable {
    let id: String
    let originalFilename: String
    let fileSize: Int64
    let dateTaken: Date
    let uploadedAt: Date
    let hash: String?
    let thumbnailUrl: String?

    init(from syncPhoto: SyncPhotoItem) {
        self.id = syncPhoto.id
        self.originalFilename = syncPhoto.originalFilename
        self.fileSize = syncPhoto.fileSize
        self.hash = syncPhoto.fileHash
        self.thumbnailUrl = syncPhoto.thumbnailUrl
        self.dateTaken = syncPhoto.dateTaken
        self.uploadedAt = syncPhoto.uploadedAt
    }

    init(from response: PhotoResponse) {
        self.id = response.id
        self.originalFilename = response.originalFilename
        self.fileSize = response.fileSize
        self.hash = response.hash
        self.thumbnailUrl = nil

        // Parse dates
        let formatter = ISO8601DateFormatter()
        self.dateTaken = formatter.date(from: response.dateTaken) ?? Date()
        self.uploadedAt = formatter.date(from: response.uploadedAt) ?? Date()
    }

    var formattedFileSize: String {
        let formatter = ByteCountFormatter()
        formatter.allowedUnits = [.useKB, .useMB, .useGB]
        formatter.countStyle = .file
        return formatter.string(fromByteCount: fileSize)
    }

    var formattedDateTaken: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: dateTaken)
    }
}

enum ServerPhotoError: Error, LocalizedError {
    case invalidImageData
    case saveFailed
    case noDeviceId

    var errorDescription: String? {
        switch self {
        case .invalidImageData:
            return "Failed to create image from downloaded data"
        case .saveFailed:
            return "Failed to save photo to library"
        case .noDeviceId:
            return "Device not registered. Please configure the app first."
        }
    }
}
