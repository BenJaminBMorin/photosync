import Foundation
import Photos
import UIKit

/// Service for managing server-only photos (photos on server but not on device)
actor ServerPhotoService {
    static let shared = ServerPhotoService()

    private let api = APIService.shared
    private let photoLibrary = PhotoLibraryService.shared

    private init() {}

    /// Get photos that exist on server but not on device
    func getServerOnlyPhotos() async throws -> [ServerPhoto] {
        await Logger.shared.info("Fetching server-only photos")

        // Fetch all local photos and compute their hashes
        let assets = await photoLibrary.fetchAllPhotos()
        var localHashes = Set<String>()

        for asset in assets {
            do {
                let imageData = try await photoLibrary.getImageData(for: asset)
                let hash = HashService.sha256(imageData)
                localHashes.insert(hash)
            } catch {
                await Logger.shared.warning("Failed to compute hash for local asset: \(error)")
            }
        }

        await Logger.shared.info("Computed \(localHashes.count) local photo hashes")

        // Fetch all photos from server
        var serverPhotos: [PhotoResponse] = []
        var skip = 0
        let take = 100
        var hasMore = true

        while hasMore {
            let response = try await api.listPhotos(skip: skip, take: take)
            serverPhotos.append(contentsOf: response.photos)

            hasMore = response.photos.count == take
            skip += take
        }

        await Logger.shared.info("Fetched \(serverPhotos.count) photos from server")

        // Filter to only server-only photos (photos with hashes not in local set)
        let serverOnlyPhotos = serverPhotos
            .filter { photo in
                guard let hash = photo.hash else { return true }
                return !localHashes.contains(hash)
            }
            .map { ServerPhoto(from: $0) }

        await Logger.shared.info("Found \(serverOnlyPhotos.count) server-only photos")

        return serverOnlyPhotos
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

/// Represents a photo that exists on the server
struct ServerPhoto: Identifiable, Codable {
    let id: String
    let originalFilename: String
    let fileSize: Int64
    let dateTaken: Date
    let uploadedAt: Date
    let hash: String?

    init(from response: PhotoResponse) {
        self.id = response.id
        self.originalFilename = response.originalFilename
        self.fileSize = response.fileSize
        self.hash = response.hash

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

    var errorDescription: String? {
        switch self {
        case .invalidImageData:
            return "Failed to create image from downloaded data"
        case .saveFailed:
            return "Failed to save photo to library"
        }
    }
}
