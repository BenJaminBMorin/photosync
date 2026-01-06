import Foundation

/// App settings stored in UserDefaults (with API key in Keychain)
struct AppSettings {
    private static let serverURLKey = "serverURL"
    private static let wifiOnlyKey = "wifiOnly"
    private static let deviceIdKey = "deviceId"
    private static let keychainMigratedKey = "keychainMigrated"

    static var serverURL: String {
        get { UserDefaults.standard.string(forKey: serverURLKey) ?? "" }
        set { UserDefaults.standard.set(newValue, forKey: serverURLKey) }
    }

    /// API key stored securely in Keychain
    static var apiKey: String {
        get { KeychainService.getAPIKey() ?? "" }
        set {
            if newValue.isEmpty {
                KeychainService.deleteAPIKey()
            } else {
                try? KeychainService.setAPIKey(newValue)
            }
        }
    }

    static var wifiOnly: Bool {
        get { UserDefaults.standard.bool(forKey: wifiOnlyKey) }
        set { UserDefaults.standard.set(newValue, forKey: wifiOnlyKey) }
    }

    static var deviceId: String? {
        get { UserDefaults.standard.string(forKey: deviceIdKey) }
        set { UserDefaults.standard.set(newValue, forKey: deviceIdKey) }
    }

    static var isConfigured: Bool {
        !serverURL.isEmpty && !apiKey.isEmpty
    }

    static var isDeviceRegistered: Bool {
        deviceId != nil
    }

    /// Normalized server URL (removes trailing slash)
    static var normalizedServerURL: String {
        var url = serverURL.trimmingCharacters(in: .whitespacesAndNewlines)
        if url.hasSuffix("/") {
            url.removeLast()
        }
        return url
    }

    /// Migrate API key from UserDefaults to Keychain (call once at app startup)
    static func performMigrations() {
        // Only run migration once
        if !UserDefaults.standard.bool(forKey: keychainMigratedKey) {
            KeychainService.migrateFromUserDefaults()
            UserDefaults.standard.set(true, forKey: keychainMigratedKey)
        }
    }
}
