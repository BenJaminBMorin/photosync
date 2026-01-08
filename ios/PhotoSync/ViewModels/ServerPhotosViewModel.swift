import Foundation
import UIKit
import SwiftUI

@MainActor
class ServerPhotosViewModel: ObservableObject {
    @Published var serverPhotos: [ServerPhotoWithThumbnail] = []
    @Published var isLoading = false
    @Published var isRestoring = false
    @Published var error: String?

    private let serverPhotoService = ServerPhotoService.shared

    func loadServerPhotos() async {
        await Logger.shared.info("Loading server-only photos")
        isLoading = true
        error = nil

        do {
            let photos = try await serverPhotoService.getServerOnlyPhotos()
            await Logger.shared.info("Found \(photos.count) server-only photos")

            // Convert to ServerPhotoWithThumbnail
            serverPhotos = photos.map { ServerPhotoWithThumbnail(photo: $0) }

            // Load thumbnails in background
            for i in serverPhotos.indices {
                Task {
                    await loadThumbnail(for: i)
                }
            }
        } catch {
            await Logger.shared.error("Failed to load server photos: \(error.localizedDescription)")
            self.error = "Failed to load server photos: \(error.localizedDescription)"
        }

        isLoading = false
    }

    private func loadThumbnail(for index: Int) async {
        guard index < serverPhotos.count else { return }

        do {
            let photo = serverPhotos[index].photo
            let thumbnail = try await serverPhotoService.downloadThumbnail(for: photo)
            serverPhotos[index].thumbnail = thumbnail
        } catch {
            await Logger.shared.warning("Failed to load thumbnail for \(serverPhotos[index].photo.id): \(error)")
        }
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
