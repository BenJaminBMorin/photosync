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
    }

    func requestAuthorization() async {
        authorizationStatus = await photoLibrary.requestAuthorization()
        if authorizationStatus == .authorized || authorizationStatus == .limited {
            await loadPhotos()
        }
    }

    func loadPhotos() async {
        isLoading = true
        error = nil

        do {
            let assets = await photoLibrary.fetchAllPhotos()
            let syncedIds = SyncedPhotoEntity.allSyncedIdentifiers(context: context)

            photos = assets.map { asset in
                let photo = Photo(asset: asset, isSynced: syncedIds.contains(asset.localIdentifier))
                return PhotoWithState(photo: photo)
            }
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
        guard !selectedPhotos.isEmpty else { return }

        syncTask = Task {
            isSyncing = true
            syncProgress = SyncProgress(total: selectedPhotos.count, completed: 0, failed: 0)

            let result = await syncService.syncPhotos(selectedPhotos, context: context) { [weak self] progress in
                Task { @MainActor in
                    self?.syncProgress = progress
                }
            }

            // Update UI
            clearSelection()
            await loadPhotos()

            isSyncing = false
            syncProgress = nil

            if result.failCount > 0 {
                error = "\(result.failCount) photos failed to sync"
            }
        }
    }

    func cancelSync() {
        syncTask?.cancel()
        isSyncing = false
        syncProgress = nil
    }

    func clearError() {
        error = nil
    }
}
