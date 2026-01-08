import Foundation

/// App settings stored in UserDefaults (with API key in Keychain)
struct AppSettings {
    private static let serverURLKey = "serverURL"
    private static let wifiOnlyKey = "wifiOnly"
    private static let deviceIdKey = "deviceId"
    private static let keychainMigratedKey = "keychainMigrated"
    private static let autoSyncKey = "autoSync"
    private static let showServerOnlyPhotosKey = "showServerOnlyPhotos"

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

    /// Auto-sync new photos in background
    static var autoSync: Bool {
        get { UserDefaults.standard.bool(forKey: autoSyncKey) }
        set {
            UserDefaults.standard.set(newValue, forKey: autoSyncKey)
            // Notify that auto-sync setting changed
            NotificationCenter.default.post(name: .autoSyncSettingChanged, object: nil)
        }
    }

    /// Show photos that are on server but not on phone
    static var showServerOnlyPhotos: Bool {
        get { UserDefaults.standard.bool(forKey: showServerOnlyPhotosKey) }
        set { UserDefaults.standard.set(newValue, forKey: showServerOnlyPhotosKey) }
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

// MARK: - Notification Names
extension Notification.Name {
    static let autoSyncSettingChanged = Notification.Name("autoSyncSettingChanged")
}
