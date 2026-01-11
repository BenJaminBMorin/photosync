import Foundation
import UIKit
import SwiftUI
import Photos

/// View model for the All Photos tab showing combined local and server photos
@MainActor
class AllPhotosViewModel: ObservableObject {
    @Published var photos: [CombinedPhoto] = []
    @Published var isLoading = false
    @Published var error: String?
    @Published var selectedPhotos: Set<String> = []

    private let serverPhotoService = ServerPhotoService.shared
    private let photoLibrary = PhotoLibraryService.shared
    private var thumbnailLoadingTasks: [String: Task<Void, Never>] = [:]
    private var loadingTask: Task<Void, Never>?
    private var isCancelled = false

    var displayedPhotos: [CombinedPhoto] {
        photos.sorted { $0.dateTaken > $1.dateTaken }
    }

    func loadAllPhotos() async {
        await Logger.shared.info("Loading all photos (local + server)")
        isLoading = true
        error = nil
        isCancelled = false

        var combinedPhotos: [CombinedPhoto] = []

        do {
            // Load local photos
            let localAssets = await photoLibrary.fetchAllPhotos()
            await Logger.shared.info("Found \(localAssets.count) local photos")

            // Check for cancellation
            if isCancelled {
                isLoading = false
                return
            }

            for asset in localAssets {
                let filename = getFilename(for: asset)
                combinedPhotos.append(CombinedPhoto(
                    id: "local_\(asset.localIdentifier)",
                    source: .local(asset),
                    dateTaken: asset.creationDate ?? Date(),
                    filename: filename
                ))
            }

            // Load server photos
            var cursor: String? = nil
            var pageCount = 0

            repeat {
                // Check for cancellation before each page
                if isCancelled {
                    isLoading = false
                    return
                }

                let page = try await serverPhotoService.getAllServerPhotos(cursor: cursor, limit: 100)

                for serverPhoto in page.photos {
                    combinedPhotos.append(CombinedPhoto(
                        id: "server_\(serverPhoto.id)",
                        source: .server(serverPhoto),
                        dateTaken: serverPhoto.dateTaken,
                        filename: serverPhoto.originalFilename
                    ))
                }

                pageCount += 1
                cursor = page.cursor

                await Logger.shared.info("Loaded page \(pageCount): \(page.photos.count) server photos")
            } while cursor != nil

            // Check for cancellation before final processing
            if isCancelled {
                isLoading = false
                return
            }

            // Remove duplicates (photos that exist both locally and on server)
            // Use a simple heuristic: if filename and date are very close, consider it a duplicate
            let uniquePhotos = removeDuplicates(from: combinedPhotos)

            photos = uniquePhotos
            await Logger.shared.info("Total unique photos: \(photos.count)")

        } catch {
            // Don't report errors if cancelled
            if !isCancelled {
                await Logger.shared.error("Failed to load all photos: \(error.localizedDescription)")
                self.error = "Failed to load photos: \(error.localizedDescription)"
            }
        }

        isLoading = false
    }

    func cancelLoading() {
        isCancelled = true
        isLoading = false
        loadingTask?.cancel()
        loadingTask = nil
    }

    func loadThumbnailIfNeeded(for photo: CombinedPhoto) {
        guard photo.thumbnail == nil,
              thumbnailLoadingTasks[photo.id] == nil else {
            return
        }

        let task = Task {
            await loadThumbnail(for: photo.id)
        }
        thumbnailLoadingTasks[photo.id] = task
    }

    private func loadThumbnail(for photoId: String) async {
        guard let index = photos.firstIndex(where: { $0.id == photoId }) else {
            thumbnailLoadingTasks.removeValue(forKey: photoId)
            return
        }

        do {
            let photo = photos[index]

            switch photo.source {
            case .local(let asset):
                let size = CGSize(width: 200, height: 200)
                if let thumbnail = try await photoLibrary.getThumbnail(for: asset, size: size) {
                    photos[index].thumbnail = thumbnail
                }

            case .server(let serverPhoto):
                let thumbnail = try await serverPhotoService.downloadThumbnail(for: serverPhoto)
                photos[index].thumbnail = thumbnail
            }
        } catch {
            await Logger.shared.warning("Failed to load thumbnail for \(photoId): \(error)")
        }

        thumbnailLoadingTasks.removeValue(forKey: photoId)
    }

    func toggleSelection(for photoId: String) {
        if selectedPhotos.contains(photoId) {
            selectedPhotos.remove(photoId)
        } else {
            selectedPhotos.insert(photoId)
        }
    }

    func clearSelection() {
        selectedPhotos = []
    }

    func clearError() {
        error = nil
    }

    private func removeDuplicates(from photos: [CombinedPhoto]) -> [CombinedPhoto] {
        var seen: [String: CombinedPhoto] = [:]
        var result: [CombinedPhoto] = []

        for photo in photos {
            let key = "\(photo.filename)_\(Int(photo.dateTaken.timeIntervalSince1970))"

            if let existing = seen[key] {
                // Keep local version if duplicate exists
                if case .local = photo.source {
                    seen[key] = photo
                }
            } else {
                seen[key] = photo
            }
        }

        result = Array(seen.values)
        return result
    }

    private func getFilename(for asset: PHAsset) -> String {
        // Try to get original filename from asset resource
        if let resource = PHAssetResource.assetResources(for: asset).first {
            return resource.originalFilename
        }

        // Fallback to date-based name
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd_HH-mm-ss"
        let dateString = formatter.string(from: asset.creationDate ?? Date())
        return "IMG_\(dateString).jpg"
    }
}

/// Represents a photo that can be from either local library or server
struct CombinedPhoto: Identifiable {
    let id: String
    let source: PhotoSource
    let dateTaken: Date
    let filename: String
    var thumbnail: UIImage?

    enum PhotoSource {
        case local(PHAsset)
        case server(ServerPhoto)
    }

    var isLocal: Bool {
        if case .local = source {
            return true
        }
        return false
    }

    var isServer: Bool {
        if case .server = source {
            return true
        }
        return false
    }
}
