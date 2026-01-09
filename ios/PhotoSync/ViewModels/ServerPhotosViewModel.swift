import Foundation
import UIKit
import SwiftUI

@MainActor
class ServerPhotosViewModel: ObservableObject {
    @Published var serverPhotos: [ServerPhotoWithThumbnail] = []
    @Published var isLoading = false
    @Published var isLoadingMore = false
    @Published var isRestoring = false
    @Published var error: String?

    private let serverPhotoService = ServerPhotoService.shared
    private var currentCursor: String?
    private var hasMore = true
    private var thumbnailLoadingTasks: [String: Task<Void, Never>] = [:]

    func loadServerPhotos() async {
        await Logger.shared.info("Loading first page of server-only photos")
        isLoading = true
        error = nil

        // Reset pagination state
        currentCursor = nil
        hasMore = true
        serverPhotos = []

        do {
            let page = try await serverPhotoService.getServerOnlyPhotos(cursor: nil, limit: 50)
            await Logger.shared.info("Found \(page.photos.count) server-only photos on first page")

            // Convert to ServerPhotoWithThumbnail
            serverPhotos = page.photos.map { ServerPhotoWithThumbnail(photo: $0) }
            currentCursor = page.cursor
            hasMore = page.hasMore

            await Logger.shared.info("Pagination: cursor=\(page.cursor ?? "nil"), hasMore=\(page.hasMore)")
        } catch {
            await Logger.shared.error("Failed to load server photos: \(error.localizedDescription)")
            self.error = "Failed to load server photos: \(error.localizedDescription)"
        }

        isLoading = false
    }

    func loadMoreIfNeeded(currentPhoto: ServerPhotoWithThumbnail) async {
        // Load more when we're near the end (within last 10 items)
        guard !isLoadingMore,
              hasMore,
              let index = serverPhotos.firstIndex(where: { $0.id == currentPhoto.id }),
              index >= serverPhotos.count - 10 else {
            return
        }

        await loadMore()
    }

    private func loadMore() async {
        guard !isLoadingMore, hasMore, let cursor = currentCursor else { return }

        await Logger.shared.info("Loading more server-only photos (cursor: \(cursor))")
        isLoadingMore = true

        do {
            let page = try await serverPhotoService.getServerOnlyPhotos(cursor: cursor, limit: 50)
            await Logger.shared.info("Loaded \(page.photos.count) more server-only photos")

            let newPhotos = page.photos.map { ServerPhotoWithThumbnail(photo: $0) }
            serverPhotos.append(contentsOf: newPhotos)
            currentCursor = page.cursor
            hasMore = page.hasMore

            await Logger.shared.info("Total photos: \(serverPhotos.count), hasMore=\(hasMore)")
        } catch {
            await Logger.shared.error("Failed to load more photos: \(error.localizedDescription)")
        }

        isLoadingMore = false
    }

    func loadThumbnailIfNeeded(for photo: ServerPhotoWithThumbnail) {
        // Don't load if already loaded or loading
        guard photo.thumbnail == nil,
              thumbnailLoadingTasks[photo.id] == nil else {
            return
        }

        // Start loading thumbnail
        let task = Task {
            await loadThumbnail(for: photo.id)
        }
        thumbnailLoadingTasks[photo.id] = task
    }

    private func loadThumbnail(for photoId: String) async {
        guard let index = serverPhotos.firstIndex(where: { $0.id == photoId }) else {
            thumbnailLoadingTasks.removeValue(forKey: photoId)
            return
        }

        do {
            let photo = serverPhotos[index].photo
            let thumbnail = try await serverPhotoService.downloadThumbnail(for: photo)
            serverPhotos[index].thumbnail = thumbnail
        } catch {
            await Logger.shared.warning("Failed to load thumbnail for \(photoId): \(error)")
        }

        thumbnailLoadingTasks.removeValue(forKey: photoId)
    }

    func restorePhoto(_ photo: ServerPhoto) async {
        await Logger.shared.info("Restoring photo: \(photo.id)")
        isRestoring = true
        error = nil

        do {
            try await serverPhotoService.restorePhoto(photo)

            // Remove from list since it's now on device
            if let index = serverPhotos.firstIndex(where: { $0.photo.id == photo.id }) {
                serverPhotos.remove(at: index)
            }

            await Logger.shared.info("Successfully restored photo: \(photo.id)")
        } catch {
            await Logger.shared.error("Failed to restore photo: \(error.localizedDescription)")
            self.error = "Failed to restore photo: \(error.localizedDescription)"
        }

        isRestoring = false
    }

    func clearError() {
        error = nil
    }
}

/// Server photo with its thumbnail
struct ServerPhotoWithThumbnail: Identifiable {
    let photo: ServerPhoto
    var thumbnail: UIImage?

    var id: String { photo.id }
}
