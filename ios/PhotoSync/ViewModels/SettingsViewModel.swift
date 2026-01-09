import Foundation

@MainActor
class SettingsViewModel: ObservableObject {
    @Published var serverURL: String {
        didSet { AppSettings.serverURL = serverURL }
    }
    @Published var apiKey: String {
        didSet { AppSettings.apiKey = apiKey }
    }
    @Published var wifiOnly: Bool {
        didSet { AppSettings.wifiOnly = wifiOnly }
    }
    @Published var autoSync: Bool {
        didSet { AppSettings.autoSync = autoSync }
    }
    @Published var showServerOnlyPhotos: Bool {
        didSet { AppSettings.showServerOnlyPhotos = showServerOnlyPhotos }
    }
    @Published var autoCleanupSyncedPhotos: Bool {
        didSet { AppSettings.autoCleanupSyncedPhotos = autoCleanupSyncedPhotos }
    }
    @Published var autoCleanupAfterDays: Int {
        didSet { AppSettings.autoCleanupAfterDays = autoCleanupAfterDays }
    }
    @Published var isTesting = false
    @Published var testResult: TestResult?
    @Published var isResyncing = false
    @Published var resyncResult: ResyncResult?
    @Published var resyncProgress: String?
    @Published var syncStatus: SyncStatusResponse?
    @Published var isClaiming = false

    // Background processing
    @Published var lastBackgroundSync: Date?
    @Published var backgroundSyncCount: Int = 0

    enum TestResult {
        case success
        case failure(String)
    }

    enum ResyncResult {
        case success(Int) // Number of photos marked as synced
        case failure(String)
    }

    private let syncService = SyncService.shared
    private let autoSyncManager = AutoSyncManager.shared

    init() {
        self.serverURL = AppSettings.serverURL
        self.apiKey = AppSettings.apiKey
        self.wifiOnly = AppSettings.wifiOnly
        self.autoSync = AppSettings.autoSync
        self.showServerOnlyPhotos = AppSettings.showServerOnlyPhotos
        self.autoCleanupSyncedPhotos = AppSettings.autoCleanupSyncedPhotos
        self.autoCleanupAfterDays = AppSettings.autoCleanupAfterDays

        // Load background sync stats
        loadBackgroundSyncStats()
    }

    var isConfigured: Bool {
        !serverURL.isEmpty && !apiKey.isEmpty
    }

    func testConnection() async {
        isTesting = true
        testResult = nil

        let result = await syncService.testConnection()

        switch result {
        case .success:
            testResult = .success
        case .failure(let error):
            testResult = .failure(error.localizedDescription)
        }

        isTesting = false
    }

    func clearTestResult() {
        testResult = nil
    }

    func resyncFromServer() async {
        isResyncing = true
        resyncResult = nil
        resyncProgress = nil

        do {
            // Fetch sync status first
            if let deviceId = AppSettings.deviceId {
                do {
                    syncStatus = try await APIService.shared.getSyncStatus(deviceId: deviceId)
                } catch {
                    await Logger.shared.error("Failed to get initial sync status: \(error.localizedDescription)")
                    // Continue anyway to try the resync
                }
            }

            try await autoSyncManager.resyncFromServer { progress in
                Task { @MainActor in
                    self.resyncProgress = progress
                }
            }

            // Refresh sync status after resync
            if let deviceId = AppSettings.deviceId {
                do {
                    syncStatus = try await APIService.shared.getSyncStatus(deviceId: deviceId)
                } catch {
                    await Logger.shared.error("Failed to get final sync status: \(error.localizedDescription)")
                    // Continue anyway since resync succeeded
                }
            }

            resyncResult = .success(0)
        } catch {
            await Logger.shared.error("Resync failed: \(error.localizedDescription)")
            resyncResult = .failure(error.localizedDescription)
        }

        isResyncing = false
        resyncProgress = nil
    }

    func clearResyncResult() {
        resyncResult = nil
    }

    func claimLegacyPhotos() async {
        guard let deviceId = AppSettings.deviceId else { return }

        isClaiming = true

        do {
            let result = try await APIService.shared.claimLegacyPhotos(deviceId: deviceId, claimAll: true)

            // Refresh sync status
            syncStatus = try? await APIService.shared.getSyncStatus(deviceId: deviceId)

            // Trigger a resync to update local state
            try? await autoSyncManager.resyncFromServer()
        } catch {
            // Handle error silently or show alert
        }

        isClaiming = false
    }

    // MARK: - Background Processing

    var isBackgroundRefreshAvailable: Bool {
        BackgroundTaskManager.shared.isBackgroundRefreshAvailable
    }

    var backgroundRefreshStatus: String {
        BackgroundTaskManager.shared.backgroundRefreshStatusString
    }

    private func loadBackgroundSyncStats() {
        lastBackgroundSync = BackgroundTaskManager.shared.lastBackgroundSyncDate
        backgroundSyncCount = BackgroundTaskManager.shared.backgroundSyncCount
    }

    func formatDate(_ date: Date) -> String {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .full
        return formatter.localizedString(for: date, relativeTo: Date())
    }
}
