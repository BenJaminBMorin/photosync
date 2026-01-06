import Foundation
import Photos
import CoreData
import Combine

@MainActor
class GalleryViewModel: ObservableObject {
    @Published var photos: [PhotoWithState] = []
    @Published var isLoading = false
    @Published var isSyncing = false
    @Published var syncProgress: SyncProgress?
    @Published var error: String?
    @Published var showUnsyncedOnly = false
    @Published var authorizationStatus: PHAuthorizationStatus = .notDetermined

    private let photoLibrary = PhotoLibraryService.shared
    private let syncService = SyncService.shared
    private var context: NSManagedObjectContext

    private var syncTask: Task<Void, Never>?

    var selectedCount: Int {
        photos.filter { $0.isSelected }.count
    }

    var syncedCount: Int {
        photos.filter { $0.syncState == .synced }.count
    }

    var unsyncedCount: Int {
        photos.filter { $0.syncState != .synced }.count
    }

    var displayedPhotos: [PhotoWithState] {
        if showUnsyncedOnly {
            return photos.filter { $0.syncState != .synced }
        }
        return photos
    }

    var isConfigured: Bool {
        AppSettings.isConfigured
    }

    init(context: NSManagedObjectContext = PersistenceController.shared.container.viewContext) {
        self.context = context
        Task {
            await Logger.shared.info("GalleryViewModel initialized")
        }
    }

    func requestAuthorization() async {
        await Logger.shared.info("Requesting photo library authorization")
        authorizationStatus = await photoLibrary.requestAuthorization()
        await Logger.shared.info("Photo library authorization status: \(authorizationStatus.rawValue)")
        if authorizationStatus == .authorized || authorizationStatus == .limited {
            await loadPhotos()
        } else {
            await Logger.shared.warning("Photo library authorization denied or restricted: \(authorizationStatus.rawValue)")
        }
    }

    func loadPhotos() async {
        await Logger.shared.info("Loading photos from library")
        isLoading = true
        error = nil

        do {
            let assets = await photoLibrary.fetchAllPhotos()
            await Logger.shared.info("Fetched \(assets.count) photos from library")

            let syncedIds = SyncedPhotoEntity.allSyncedIdentifiers(context: context)
            await Logger.shared.info("Found \(syncedIds.count) synced photos in database")

            photos = assets.map { asset in
                let photo = Photo(asset: asset, isSynced: syncedIds.contains(asset.localIdentifier))
                return PhotoWithState(photo: photo)
            }

            await Logger.shared.info("Loaded \(photos.count) photos (synced: \(syncedCount), unsynced: \(unsyncedCount))")
        } catch {
            await Logger.shared.error("Failed to load photos: \(error.localizedDescription)")
            self.error = "Failed to load photos: \(error.localizedDescription)"
        }

        isLoading = false
    }

    func toggleSelection(for photoId: String) {
        if let index = photos.firstIndex(where: { $0.id == photoId }) {
            photos[index].isSelected.toggle()
        }
    }

    func selectAll() {
        for i in photos.indices {
            if photos[i].syncState != .synced {
                photos[i].isSelected = true
            }
        }
    }

    func clearSelection() {
        for i in photos.indices {
            photos[i].isSelected = false
        }
    }

    func toggleUnsyncedFilter() {
        showUnsyncedOnly.toggle()
    }

    func syncSelected() {
        let selectedPhotos = photos.filter { $0.isSelected }.map { $0.photo }
        guard !selectedPhotos.isEmpty else {
            Task {
                await Logger.shared.warning("syncSelected called with no photos selected")
            }
            return
        }

        Task {
            await Logger.shared.info("Starting sync for \(selectedPhotos.count) selected photos")
        }

        syncTask = Task {
            isSyncing = true
            syncProgress = SyncProgress(total: selectedPhotos.count, completed: 0, failed: 0)

            do {
                let result = await syncService.syncPhotos(selectedPhotos, context: context) { [weak self] progress in
                    Task { @MainActor in
                        self?.syncProgress = progress
                    }
                }

                await Logger.shared.info("Sync completed: \(result.successCount) succeeded, \(result.failCount) failed")

                // Update UI
                clearSelection()
                await loadPhotos()

                isSyncing = false
                syncProgress = nil

                if result.failCount > 0 {
                    error = "\(result.failCount) photos failed to sync"
                    await Logger.shared.error("Sync errors: \(result.failCount) photos failed")
                }
            } catch {
                await Logger.shared.error("Sync task failed with error: \(error.localizedDescription)")
                self.error = "Sync failed: \(error.localizedDescription)"
                isSyncing = false
                syncProgress = nil
            }
        }
    }

    func cancelSync() {
        Task {
            await Logger.shared.info("Sync cancelled by user")
        }
        syncTask?.cancel()
        isSyncing = false
        syncProgress = nil
    }

    func clearError() {
        error = nil
    }
}
