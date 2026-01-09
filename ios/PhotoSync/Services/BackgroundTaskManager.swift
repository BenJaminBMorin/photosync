import Foundation
import BackgroundTasks
import UIKit

/// Manages background task scheduling and execution for photo syncing
@MainActor
class BackgroundTaskManager: ObservableObject {
    static let shared = BackgroundTaskManager()

    // Background task identifiers - must match Info.plist
    static let photoSyncTaskIdentifier = "com.morinclan.photosync.sync"
    static let refreshTaskIdentifier = "com.morinclan.photosync.refresh"

    @Published var lastBackgroundSyncDate: Date?
    @Published var backgroundSyncCount = 0

    private init() {
        loadState()
    }

    // MARK: - Task Registration

    /// Register all background tasks with the system
    /// Must be called early in app launch, before app finishes launching
    func registerBackgroundTasks() {
        // Register processing task for full photo sync (can run for several minutes)
        BGTaskScheduler.shared.register(
            forTaskWithIdentifier: Self.photoSyncTaskIdentifier,
            using: nil
        ) { task in
            Task { @MainActor in
                await self.handlePhotoSyncTask(task: task as! BGProcessingTask)
            }
        }

        // Register refresh task for quick checks (runs for ~30 seconds)
        BGTaskScheduler.shared.register(
            forTaskWithIdentifier: Self.refreshTaskIdentifier,
            using: nil
        ) { task in
            Task { @MainActor in
                await self.handleRefreshTask(task: task as! BGAppRefreshTask)
            }
        }

        await Logger.shared.info("Background tasks registered")
    }

    // MARK: - Task Scheduling

    /// Schedule background photo sync task
    func schedulePhotoSyncTask() {
        let request = BGProcessingTaskRequest(identifier: Self.photoSyncTaskIdentifier)

        // Require wifi and external power for large sync operations
        request.requiresNetworkConnectivity = true
        request.requiresExternalPower = false // Allow on battery, but prefer charging

        // Try to run as soon as possible, but system will decide based on conditions
        request.earliestBeginDate = Date(timeIntervalSinceNow: 15 * 60) // 15 minutes from now

        do {
            try BGTaskScheduler.shared.submit(request)
            await Logger.shared.info("Scheduled background photo sync task")
        } catch {
            await Logger.shared.info("Failed to schedule photo sync task: \(error.localizedDescription)")
        }
    }

    /// Schedule background refresh task (lighter weight, more frequent)
    func scheduleRefreshTask() {
        let request = BGAppRefreshTaskRequest(identifier: Self.refreshTaskIdentifier)

        // Can run more frequently, but for shorter duration
        request.earliestBeginDate = Date(timeIntervalSinceNow: 15 * 60) // 15 minutes minimum

        do {
            try BGTaskScheduler.shared.submit(request)
            await Logger.shared.info("Scheduled background refresh task")
        } catch {
            await Logger.shared.info("Failed to schedule refresh task: \(error.localizedDescription)")
        }
    }

    /// Cancel all pending background tasks
    func cancelAllTasks() {
        BGTaskScheduler.shared.cancel(taskRequestWithIdentifier: Self.photoSyncTaskIdentifier)
        BGTaskScheduler.shared.cancel(taskRequestWithIdentifier: Self.refreshTaskIdentifier)
        await Logger.shared.info("Cancelled all background tasks")
    }

    // MARK: - Task Handlers

    /// Handle full photo sync in background (can run for several minutes)
    private func handlePhotoSyncTask(task: BGProcessingTask) async {
        await Logger.shared.info("Background photo sync task started")

        // Schedule the next sync before we start this one
        schedulePhotoSyncTask()

        // Set up expiration handler
        task.expirationHandler = {
            Task { @MainActor in
                await Logger.shared.info("Background photo sync task expired")
                // Cancel any ongoing operations
                await AutoSyncManager.shared.cancelSync()
            }
        }

        // Only sync if auto-sync is enabled
        guard AppSettings.autoSync else {
            await Logger.shared.info("Auto-sync disabled, skipping background sync")
            task.setTaskCompleted(success: true)
            return
        }

        // Perform the sync
        do {
            await Logger.shared.info("Starting background photo sync")
            await AutoSyncManager.shared.performBackgroundSync()

            // Update state
            lastBackgroundSyncDate = Date()
            backgroundSyncCount += 1
            saveState()

            await Logger.shared.info("Background photo sync completed successfully")
            task.setTaskCompleted(success: true)
        } catch {
            await Logger.shared.error("Background photo sync failed: \(error.localizedDescription)")
            task.setTaskCompleted(success: false)
        }
    }

    /// Handle quick refresh in background (~30 seconds max)
    private func handleRefreshTask(task: BGAppRefreshTask) async {
        await Logger.shared.info("Background refresh task started")

        // Schedule the next refresh
        scheduleRefreshTask()

        // Set up expiration handler
        task.expirationHandler = {
            Task { @MainActor in
                await Logger.shared.info("Background refresh task expired")
            }
        }

        // Only refresh if auto-sync is enabled
        guard AppSettings.autoSync else {
            task.setTaskCompleted(success: true)
            return
        }

        // Quick check for new photos and sync if needed
        do {
            let hasNewPhotos = await AutoSyncManager.shared.hasUnsyncedPhotos()

            if hasNewPhotos {
                await Logger.shared.info("New photos detected, performing quick sync")
                await AutoSyncManager.shared.performBackgroundSync()

                lastBackgroundSyncDate = Date()
                backgroundSyncCount += 1
                saveState()
            } else {
                await Logger.shared.info("No new photos to sync")
            }

            task.setTaskCompleted(success: true)
        } catch {
            await Logger.shared.error("Background refresh failed: \(error.localizedDescription)")
            task.setTaskCompleted(success: false)
        }
    }

    // MARK: - State Persistence

    private func saveState() {
        UserDefaults.standard.set(lastBackgroundSyncDate, forKey: "lastBackgroundSyncDate")
        UserDefaults.standard.set(backgroundSyncCount, forKey: "backgroundSyncCount")
    }

    private func loadState() {
        lastBackgroundSyncDate = UserDefaults.standard.object(forKey: "lastBackgroundSyncDate") as? Date
        backgroundSyncCount = UserDefaults.standard.integer(forKey: "backgroundSyncCount")
    }

    // MARK: - Public Helpers

    /// Enable background syncing (schedules tasks)
    func enableBackgroundSync() {
        schedulePhotoSyncTask()
        scheduleRefreshTask()
        await Logger.shared.info("Background sync enabled")
    }

    /// Disable background syncing (cancels tasks)
    func disableBackgroundSync() {
        cancelAllTasks()
        await Logger.shared.info("Background sync disabled")
    }

    /// Check if background refresh is available
    var isBackgroundRefreshAvailable: Bool {
        UIApplication.shared.backgroundRefreshStatus == .available
    }

    /// Get human-readable background refresh status
    var backgroundRefreshStatusString: String {
        switch UIApplication.shared.backgroundRefreshStatus {
        case .available:
            return "Available"
        case .denied:
            return "Denied - Enable in Settings"
        case .restricted:
            return "Restricted"
        @unknown default:
            return "Unknown"
        }
    }
}
