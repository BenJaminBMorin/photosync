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
    @Published var showIgnoredPhotos = false
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
        photos.filter { $0.syncState != .synced && $0.syncState != .ignored }.count
    }

    var ignoredCount: Int {
        photos.filter { $0.syncState == .ignored }.count
    }

    var displayedPhotos: [PhotoWithState] {
        var filtered = photos

        // Filter out ignored photos unless we're showing them
        if !showIgnoredPhotos {
            filtered = filtered.filter { $0.syncState != .ignored }
        }

        // Filter unsynced only
        if showUnsyncedOnly {
            filtered = filtered.filter { $0.syncState != .synced }
        }

        return filtered
    }

    var groupedPhotos: [PhotoGroup] {
        let filtered = displayedPhotos

        // Group by year and month
        let grouped = Dictionary(grouping: filtered) { photo -> String in
            let components = Calendar.current.dateComponents([.year, .month], from: photo.photo.creationDate)
            return String(format: "%04d-%02d", components.year ?? 0, components.month ?? 0)
        }

        // Convert to PhotoGroup array and sort
        return grouped.map { key, photos in
            let components = key.split(separator: "-")
            let year = Int(components[0]) ?? 0
            let month = Int(components[1]) ?? 0
            return PhotoGroup(id: key, year: year, month: month, photos: photos)
        }
        .sorted { $0.year == $1.year ? $0.month > $1.month : $0.year > $1.year }
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

            let ignoredIds = IgnoredPhotoEntity.allIgnoredIdentifiers(context: context)
            await Logger.shared.info("Found \(ignoredIds.count) ignored photos in database")

            photos = assets.map { asset in
                let photo = Photo(asset: asset, isSynced: syncedIds.contains(asset.localIdentifier))
                let isIgnored = ignoredIds.contains(asset.localIdentifier)
                var photoWithState = PhotoWithState(photo: photo)
                if isIgnored {
                    photoWithState.syncState = .ignored
                }
                return photoWithState
            }

            await Logger.shared.info("Loaded \(photos.count) photos (synced: \(syncedCount), unsynced: \(unsyncedCount), ignored: \(ignoredCount))")
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
            if photos[i].syncState != .synced && photos[i].syncState != .ignored {
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

    func toggleIgnoredFilter() {
        showIgnoredPhotos.toggle()
    }

    func toggleIgnore(for photoId: String) {
        guard let index = photos.firstIndex(where: { $0.id == photoId }) else { return }

        let photo = photos[index]

        if photo.syncState == .ignored {
            // Unignore the photo
            IgnoredPhotoEntity.unignore(localIdentifier: photo.photo.localIdentifier, context: context)
            photos[index].syncState = photo.photo.isSynced ? .synced : .notSynced
            Task {
                await Logger.shared.info("Unignored photo: \(photo.id)")
            }
        } else {
            // Ignore the photo
            _ = IgnoredPhotoEntity.create(context: context, localIdentifier: photo.photo.localIdentifier)
            do {
                try context.save()
                photos[index].syncState = .ignored
                photos[index].isSelected = false  // Deselect ignored photos
                Task {
                    await Logger.shared.info("Ignored photo: \(photo.id)")
                }
            } catch {
                Task {
                    await Logger.shared.error("Failed to ignore photo: \(error.localizedDescription)")
                }
            }
        }
    }

    var selectedIgnoredCount: Int {
        photos.filter { $0.isSelected && $0.syncState == .ignored }.count
    }

    var selectedNonIgnoredCount: Int {
        photos.filter { $0.isSelected && $0.syncState != .ignored }.count
    }

    func ignoreSelected() {
        let selectedIndices = photos.indices.filter { photos[$0].isSelected && photos[$0].syncState != .ignored }
        guard !selectedIndices.isEmpty else { return }

        Task {
            await Logger.shared.info("Ignoring \(selectedIndices.count) selected photos")
        }

        for index in selectedIndices {
            let photo = photos[index]
            _ = IgnoredPhotoEntity.create(context: context, localIdentifier: photo.photo.localIdentifier)
            photos[index].syncState = .ignored
            photos[index].isSelected = false
        }

        do {
            try context.save()
            Task {
                await Logger.shared.info("Successfully ignored \(selectedIndices.count) photos")
            }
        } catch {
            Task {
                await Logger.shared.error("Failed to ignore selected photos: \(error.localizedDescription)")
            }
        }
    }

    func unignoreSelected() {
        let selectedIndices = photos.indices.filter { photos[$0].isSelected && photos[$0].syncState == .ignored }
        guard !selectedIndices.isEmpty else { return }

        Task {
            await Logger.shared.info("Unignoring \(selectedIndices.count) selected photos")
        }

        for index in selectedIndices {
            let photo = photos[index]
            IgnoredPhotoEntity.unignore(localIdentifier: photo.photo.localIdentifier, context: context)
            photos[index].syncState = photo.photo.isSynced ? .synced : .notSynced
            photos[index].isSelected = false
        }

        Task {
            await Logger.shared.info("Successfully unignored \(selectedIndices.count) photos")
        }
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
                let result = await syncService.syncPhotos(selectedPhotos, context: context) { progress in
                    Task { @MainActor [weak self] in
                        guard let self else { return }
                        // Only update if this is newer progress (using sequence number)
                        if self.syncProgress == nil || progress.sequence > (self.syncProgress?.sequence ?? -1) {
                            self.syncProgress = progress
                        }
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
