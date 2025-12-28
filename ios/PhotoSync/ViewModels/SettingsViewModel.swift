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
    @Published var isTesting = false
    @Published var testResult: TestResult?

    enum TestResult {
        case success
        case failure(String)
    }

    private let syncService = SyncService.shared

    init() {
        self.serverURL = AppSettings.serverURL
        self.apiKey = AppSettings.apiKey
        self.wifiOnly = AppSettings.wifiOnly
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
}
