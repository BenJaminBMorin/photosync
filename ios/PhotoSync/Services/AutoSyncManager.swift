import Foundation
import Photos
import CoreData
import Network

/// Manages automatic background syncing of new photos
@MainActor
class AutoSyncManager: ObservableObject {
    static let shared = AutoSyncManager()

    @Published var isAutoSyncing = false
    @Published var autoSyncProgress: SyncProgress?

    private let photoLibrary = PhotoLibraryService.shared
    private let syncService = SyncService.shared
    private let context: NSManagedObjectContext
    private let networkMonitor = NWPathMonitor()
    private let monitorQueue = DispatchQueue(label: "com.photosync.network-monitor")

    private var isWifiConnected = false
    private var autoSyncTask: Task<Void, Never>?
    private var photoLibraryObserver: NSObjectProtocol?

    private init() {
        self.context = PersistenceController.shared.container.viewContext

        // Start network monitoring
        setupNetworkMonitoring()

        // Listen for auto-sync setting changes
        NotificationCenter.default.addObserver(
            forName: .autoSyncSettingChanged,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.handleAutoSyncSettingChanged()
            }
        }

        // Listen for photo library changes
        setupPhotoLibraryObserver()

        // Start auto-sync if enabled
        if AppSettings.autoSync {
            Task {
                await startAutoSyncIfNeeded()
            }
        }
    }

    deinit {
        networkMonitor.cancel()
        if let observer = photoLibraryObserver {
            NotificationCenter.default.removeObserver(observer)
        }
    }

    // MARK: - Network Monitoring

    private func setupNetworkMonitoring() {
        networkMonitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor [weak self] in
                guard let self else { return }

                let wasWifi = self.isWifiConnected
                self.isWifiConnected = path.usesInterfaceType(.wifi)

                // If wifi just connected and auto-sync is enabled, trigger sync
                if !wasWifi && self.isWifiConnected && AppSettings.autoSync {
                    await self.startAutoSyncIfNeeded()
                }
            }
        }
        networkMonitor.start(queue: monitorQueue)
    }

    // MARK: - Photo Library Monitoring

    private func setupPhotoLibraryObserver() {
        photoLibraryObserver = NotificationCenter.default.addObserver(
            forName: .collectionDidChange,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self else { return }
                if AppSettings.autoSync {
                    await self.startAutoSyncIfNeeded()
                }
            }
        }
    }

    // MARK: - Auto-Sync Control

    private func handleAutoSyncSettingChanged() async {
        await Logger.shared.info("Auto-sync setting changed to: \(AppSettings.autoSync)")

        if AppSettings.autoSync {
            await startAutoSyncIfNeeded()
        } else {
            cancelAutoSync()
        }
    }

    func startAutoSyncIfNeeded() async {
        // Don't start if already syncing
        guard !isAutoSyncing else {
            await Logger.shared.info("Auto-sync already in progress, skipping")
            return
        }

        // Check if auto-sync is enabled
        guard AppSettings.autoSync else {
            await Logger.shared.info("Auto-sync is disabled")
            return
        }

        // Check wifi requirement
        if AppSettings.wifiOnly && !isWifiConnected {
            await Logger.shared.info("Auto-sync requires wifi, but not connected")
            return
        }

        // Check if configured
        guard AppSettings.isConfigured else {
            await Logger.shared.warning("Auto-sync attempted but app not configured")
            return
        }

        await Logger.shared.info("Starting auto-sync...")

        // Get unsynced photos
        let assets = await photoLibrary.fetchAllPhotos()
        let syncedIds = SyncedPhotoEntity.allSyncedIdentifiers(context: context)
        let ignoredIds = IgnoredPhotoEntity.allIgnoredIdentifiers(context: context)

        let unsyncedPhotos = assets
            .filter { asset in
                !syncedIds.contains(asset.localIdentifier) &&
                !ignoredIds.contains(asset.localIdentifier)
            }
            .map { Photo(asset: $0, isSynced: false) }

        guard !unsyncedPhotos.isEmpty else {
            await Logger.shared.info("No unsynced photos found")
            return
        }

        await Logger.shared.info("Found \(unsyncedPhotos.count) unsynced photos for auto-sync")

        // Start syncing
        autoSyncTask = Task {
            isAutoSyncing = true
            autoSyncProgress = SyncProgress(
                total: unsyncedPhotos.count,
                completed: 0,
                failed: 0,
                sequence: 0
            )

            let result = await syncService.syncPhotos(unsyncedPhotos, context: context) { [weak self] progress in
                Task { @MainActor [weak self] in
                    guard let self else { return }
                    self.autoSyncProgress = progress
                }
            }

            await Logger.shared.info("Auto-sync completed: \(result.successCount) succeeded, \(result.failCount) failed")

            isAutoSyncing = false
            autoSyncProgress = nil

            // Run auto-cleanup if enabled
            if AppSettings.autoCleanupSyncedPhotos {
                await performAutoCleanup()
            }

            // Notify that photos were synced
            NotificationCenter.default.post(name: .collectionDidChange, object: nil)
        }
    }

    func cancelAutoSync() {
        autoSyncTask?.cancel()
        isAutoSyncing = false
        autoSyncProgress = nil
        Task {
            await Logger.shared.info("Auto-sync cancelled")
        }
    }

    // MARK: - Database Resync

    /// Resync the local database from the server
    /// This fetches all photos on the server and marks matching local photos as synced
    func resyncFromServer() async throws {
        await Logger.shared.info("Starting database resync from server")

        guard AppSettings.isConfigured else {
            throw AutoSyncError.notConfigured
        }

        // Fetch all local photos
        let assets = await photoLibrary.fetchAllPhotos()
        await Logger.shared.info("Found \(assets.count) local photos")

        // Compute hashes for all local photos
        var localHashes: [String: String] = [:] // hash -> localIdentifier
        for asset in assets {
            do {
                let imageData = try await photoLibrary.getImageData(for: asset)
                let hash = HashService.sha256(imageData)
                localHashes[hash] = asset.localIdentifier
            } catch {
                await Logger.shared.warning("Failed to compute hash for asset: \(error)")
            }
        }

        await Logger.shared.info("Computed \(localHashes.count) hashes")

        // Fetch all photos from server in batches
        var allServerHashes: Set<String> = []
        var skip = 0
        let take = 100
        var hasMore = true

        while hasMore {
            let response = try await APIService.shared.listPhotos(skip: skip, take: take)

            for photo in response.photos {
                if let hash = photo.hash {
                    allServerHashes.insert(hash)
                }
            }

            hasMore = response.photos.count == take
            skip += take

            await Logger.shared.info("Fetched \(response.photos.count) photos from server (total: \(allServerHashes.count))")
        }

        await Logger.shared.info("Found \(allServerHashes.count) photos on server")

        // Mark matching photos as synced
        var markedCount = 0
        for (hash, localIdentifier) in localHashes {
            if allServerHashes.contains(hash) {
                // Check if already marked as synced
                if !SyncedPhotoEntity.isSynced(localIdentifier: localIdentifier, context: context) {
                    _ = SyncedPhotoEntity.create(
                        context: context,
                        localIdentifier: localIdentifier,
                        serverPhotoId: hash, // Use hash as placeholder ID
                        displayName: "",
                        dateTaken: Date()
                    )
                    markedCount += 1
                }
            }
        }

        // Save changes
        if markedCount > 0 {
            try context.save()
            await Logger.shared.info("Marked \(markedCount) photos as synced")

            // Notify UI to refresh
            NotificationCenter.default.post(name: .collectionDidChange, object: nil)
        } else {
            await Logger.shared.info("No new photos to mark as synced")
        }
    }

    // MARK: - Auto-Cleanup

    /// Automatically cleanup synced photos that are older than the configured days
    private func performAutoCleanup() async {
        await Logger.shared.info("Starting auto-cleanup of synced photos...")

        let cleanupAfterDays = AppSettings.autoCleanupAfterDays
        await Logger.shared.info("Cleanup threshold: \(cleanupAfterDays) days")

        // Calculate the cutoff date
        guard let cutoffDate = Calendar.current.date(byAdding: .day, value: -cleanupAfterDays, to: Date()) else {
            await Logger.shared.error("Failed to calculate cutoff date")
            return
        }

        await Logger.shared.info("Photos older than \(cutoffDate) will be considered for cleanup")

        // Get all synced photo identifiers
        let syncedIds = SyncedPhotoEntity.allSyncedIdentifiers(context: context)
        await Logger.shared.info("Found \(syncedIds.count) synced photos in database")

        // Fetch all local photos
        let assets = await photoLibrary.fetchAllPhotos()

        // Filter for synced photos older than cutoff date
        let photosToCleanup = assets.filter { asset in
            // Must be synced
            guard syncedIds.contains(asset.localIdentifier) else { return false }

            // Must be older than cutoff date
            guard let creationDate = asset.creationDate else { return false }
            return creationDate < cutoffDate
        }

        guard !photosToCleanup.isEmpty else {
            await Logger.shared.info("No photos eligible for auto-cleanup")
            return
        }

        await Logger.shared.info("Found \(photosToCleanup.count) photos eligible for cleanup")

        // Delete photos one by one
        var deletedCount = 0
        var failedCount = 0

        for asset in photosToCleanup {
            do {
                try await PHPhotoLibrary.shared().performChanges {
                    PHAssetChangeRequest.deleteAssets([asset] as NSArray)
                }
                deletedCount += 1
                await Logger.shared.info("Auto-deleted photo: \(asset.localIdentifier)")
            } catch {
                failedCount += 1
                await Logger.shared.error("Failed to auto-delete photo \(asset.localIdentifier): \(error)")
            }
        }

        await Logger.shared.info("Auto-cleanup complete: \(deletedCount) deleted, \(failedCount) failed")

        // Notify UI to refresh
        if deletedCount > 0 {
            NotificationCenter.default.post(name: .collectionDidChange, object: nil)
        }
    }
}

enum AutoSyncError: Error, LocalizedError {
    case notConfigured

    var errorDescription: String? {
        switch self {
        case .notConfigured:
            return "App not configured with server URL and API key"
        }
    }
}
