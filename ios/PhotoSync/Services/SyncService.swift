import Foundation
import Photos
import CoreData

/// Service for syncing photos to the server
actor SyncService {
    static let shared = SyncService()

    private let photoLibrary = PhotoLibraryService.shared
    private let api = APIService.shared

    private init() {}

    /// Sync a batch of photos
    func syncPhotos(
        _ photos: [Photo],
        context: NSManagedObjectContext,
        progressHandler: @escaping (SyncProgress) -> Void
    ) async -> SyncResult {
        var progress = SyncProgress(total: photos.count, completed: 0, failed: 0)
        var successfulUploads: [String] = []
        var errors: [String: Error] = [:]

        for photo in photos {
            if Task.isCancelled {
                progress.isCancelled = true
                progressHandler(progress)
                break
            }

            let filename = await photoLibrary.getFilename(for: photo.asset)
            progress.currentFileName = filename
            progressHandler(progress)

            do {
                let imageData = try await photoLibrary.getImageData(for: photo.asset)
                let response = try await api.uploadPhoto(
                    imageData: imageData,
                    filename: filename,
                    dateTaken: photo.creationDate
                )

                // Save to Core Data
                await context.perform {
                    _ = SyncedPhotoEntity.create(
                        context: context,
                        localIdentifier: photo.localIdentifier,
                        serverPhotoId: response.id,
                        displayName: filename,
                        dateTaken: photo.creationDate
                    )

                    do {
                        try context.save()
                    } catch {
                        print("Failed to save synced photo: \(error)")
                    }
                }

                successfulUploads.append(photo.id)
                progress.completed += 1
            } catch {
                print("Failed to sync \(filename): \(error)")
                errors[photo.id] = error
                progress.failed += 1
            }

            progressHandler(progress)
        }

        return SyncResult(
            successful: successfulUploads,
            failed: errors,
            wasCancelled: progress.isCancelled
        )
    }

    /// Test server connection
    func testConnection() async -> Result<Void, Error> {
        do {
            _ = try await api.healthCheck()
            return .success(())
        } catch {
            return .failure(error)
        }
    }
}

struct SyncResult {
    let successful: [String]
    let failed: [String: Error]
    let wasCancelled: Bool

    var successCount: Int { successful.count }
    var failCount: Int { failed.count }
    var totalProcessed: Int { successCount + failCount }
}
