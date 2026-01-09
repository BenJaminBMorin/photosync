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
    @Published var showNotOnDeviceOnly = false  // Filter toggle
    @Published var selectedPhotos: Set<String> = []  // For multi-select
    @Published var collections: [ServerCollection] = []  // Available collections
    @Published var isLoadingCollections = false

    private let serverPhotoService = ServerPhotoService.shared
    private let api = APIService.shared
    private var currentCursor: String?
    private var hasMore = true
    private var thumbnailLoadingTasks: [String: Task<Void, Never>] = [:]

    var displayedPhotos: [ServerPhotoWithThumbnail] {
        if showNotOnDeviceOnly {
            return serverPhotos.filter { !$0.photo.isOnDevice }
        } else {
            return serverPhotos
        }
    }

    var notOnDeviceCount: Int {
        serverPhotos.filter { !$0.photo.isOnDevice }.count
    }

    var onDeviceCount: Int {
        serverPhotos.filter { $0.photo.isOnDevice }.count
    }

    func loadServerPhotos() async {
        await Logger.shared.info("Loading first page of all server photos")
        isLoading = true
        error = nil

        // Reset pagination state
        currentCursor = nil
        hasMore = true
        serverPhotos = []
        selectedPhotos = []

        do {
            let page = try await serverPhotoService.getAllServerPhotos(cursor: nil, limit: 50)
            await Logger.shared.info("Found \(page.photos.count) server photos on first page")

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
              let index = displayedPhotos.firstIndex(where: { $0.id == currentPhoto.id }),
              index >= displayedPhotos.count - 10 else {
            return
        }

        await loadMore()
    }

    private func loadMore() async {
        guard !isLoadingMore, hasMore, let cursor = currentCursor else { return }

        await Logger.shared.info("Loading more server photos (cursor: \(cursor))")
        isLoadingMore = true

        do {
            let page = try await serverPhotoService.getAllServerPhotos(cursor: cursor, limit: 50)
            await Logger.shared.info("Loaded \(page.photos.count) more server photos")

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

    func restorePhoto(_ photo: ServerPhoto) async {
        await Logger.shared.info("Restoring photo: \(photo.id)")
        isRestoring = true
        error = nil

        do {
            try await serverPhotoService.restorePhoto(photo)

            // Update photo status instead of removing it
            if let index = serverPhotos.firstIndex(where: { $0.photo.id == photo.id }) {
                // Photo is now on device, update the status
                // We'll need to reload to get updated device status
                await Logger.shared.info("Photo restored, reloading to update status")
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

    func deleteSelectedPhotos() async {
        let photoIds = Array(selectedPhotos)
        guard !photoIds.isEmpty else { return }

        await Logger.shared.info("Deleting \(photoIds.count) selected photos from server")
        error = nil

        do {
            try await serverPhotoService.deletePhotos(photoIds)

            // Remove deleted photos from the list
            serverPhotos.removeAll { selectedPhotos.contains($0.id) }

            // Clear selection
            selectedPhotos = []

            await Logger.shared.info("Successfully deleted \(photoIds.count) photos")
        } catch {
            await Logger.shared.error("Failed to delete photos: \(error.localizedDescription)")
            self.error = "Failed to delete photos: \(error.localizedDescription)"
        }
    }

    // MARK: - Collections Management

    func loadCollections() async {
        await Logger.shared.info("Loading collections from server")
        isLoadingCollections = true
        error = nil

        do {
            let response = try await api.getCollections()
            collections = response.collections
            await Logger.shared.info("Loaded \(collections.count) collections")
        } catch {
            await Logger.shared.error("Failed to load collections: \(error.localizedDescription)")
            self.error = "Failed to load collections: \(error.localizedDescription)"
        }

        isLoadingCollections = false
    }

    func createCollection(name: String) async -> Bool {
        await Logger.shared.info("Creating collection: \(name)")
        error = nil

        do {
            let response = try await api.createCollection(name: name)
            collections.append(response.collection)
            await Logger.shared.info("Successfully created collection: \(name)")
            return true
        } catch {
            await Logger.shared.error("Failed to create collection: \(error.localizedDescription)")
            self.error = "Failed to create collection: \(error.localizedDescription)"
            return false
        }
    }

    func deleteCollection(_ collection: ServerCollection) async {
        await Logger.shared.info("Deleting collection: \(collection.name)")
        error = nil

        do {
            try await api.deleteCollection(collectionId: collection.id)
            collections.removeAll { $0.id == collection.id }
            await Logger.shared.info("Successfully deleted collection: \(collection.name)")
        } catch {
            await Logger.shared.error("Failed to delete collection: \(error.localizedDescription)")
            self.error = "Failed to delete collection: \(error.localizedDescription)"
        }
    }

    func addSelectedPhotosToCollection(_ collection: ServerCollection) async {
        let photoIds = Array(selectedPhotos)
        guard !photoIds.isEmpty else { return }

        await Logger.shared.info("Adding \(photoIds.count) photos to collection: \(collection.name)")
        error = nil

        do {
            try await api.addPhotosToCollection(collectionId: collection.id, photoIds: photoIds)
            await Logger.shared.info("Successfully added \(photoIds.count) photos to collection")

            // Clear selection after adding to collection
            selectedPhotos = []
        } catch {
            await Logger.shared.error("Failed to add photos to collection: \(error.localizedDescription)")
            self.error = "Failed to add photos to collection: \(error.localizedDescription)"
        }
    }
}

/// Server photo with its thumbnail
struct ServerPhotoWithThumbnail: Identifiable {
    let photo: ServerPhoto
    var thumbnail: UIImage?

    var id: String { photo.id }
}
