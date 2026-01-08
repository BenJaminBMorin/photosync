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
    @Published var isTesting = false
    @Published var testResult: TestResult?
    @Published var isResyncing = false
    @Published var resyncResult: ResyncResult?

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

        do {
            try await autoSyncManager.resyncFromServer()
            // Count how many photos are now marked as synced
            // This is a rough estimate since we don't track the count from resync
            resyncResult = .success(0)
        } catch {
            resyncResult = .failure(error.localizedDescription)
        }

        isResyncing = false
    }

    func clearResyncResult() {
        resyncResult = nil
    }
}
