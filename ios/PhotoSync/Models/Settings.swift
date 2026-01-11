import Foundation

/// App settings stored in UserDefaults (with API key in Keychain)
struct AppSettings {
    private static let serverURLKey = "serverURL"
    private static let wifiOnlyKey = "wifiOnly"
    private static let deviceIdKey = "deviceId"
    private static let keychainMigratedKey = "keychainMigrated"
    private static let autoSyncKey = "autoSync"
    private static let showServerOnlyPhotosKey = "showServerOnlyPhotos"
    private static let autoCleanupSyncedPhotosKey = "autoCleanupSyncedPhotos"
    private static let autoCleanupAfterDaysKey = "autoCleanupAfterDays"
    private static let userEmailKey = "userEmail"

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

    /// Automatically cleanup synced photos from device
    static var autoCleanupSyncedPhotos: Bool {
        get { UserDefaults.standard.bool(forKey: autoCleanupSyncedPhotosKey) }
        set { UserDefaults.standard.set(newValue, forKey: autoCleanupSyncedPhotosKey) }
    }

    /// Days to wait before auto-cleanup (default: 30)
    static var autoCleanupAfterDays: Int {
        get {
            let days = UserDefaults.standard.integer(forKey: autoCleanupAfterDaysKey)
            return days > 0 ? days : 30 // Default to 30 days
        }
        set { UserDefaults.standard.set(newValue, forKey: autoCleanupAfterDaysKey) }
    }

    static var deviceId: String? {
        get { UserDefaults.standard.string(forKey: deviceIdKey) }
        set { UserDefaults.standard.set(newValue, forKey: deviceIdKey) }
    }

    /// Email of the currently logged-in user
    static var userEmail: String? {
        get { UserDefaults.standard.string(forKey: userEmailKey) }
        set { UserDefaults.standard.set(newValue, forKey: userEmailKey) }
    }

    static var isConfigured: Bool {
        !serverURL.isEmpty && !apiKey.isEmpty
    }

    /// Server URL is set but user hasn't authenticated yet
    static var hasServerButNotAuthenticated: Bool {
        !serverURL.isEmpty && apiKey.isEmpty
    }

    static var isDeviceRegistered: Bool {
        deviceId != nil
    }

    /// Check if user needs to authenticate (no API key stored)
    static var needsAuthentication: Bool {
        apiKey.isEmpty
    }

    /// Timestamp of last successful authentication (to prevent race conditions)
    private static let lastAuthTimeKey = "photosync_last_auth_time"

    static var lastAuthenticatedAt: Date? {
        get { UserDefaults.standard.object(forKey: lastAuthTimeKey) as? Date }
        set { UserDefaults.standard.set(newValue, forKey: lastAuthTimeKey) }
    }

    /// Record that authentication just happened
    static func recordAuthentication() {
        lastAuthenticatedAt = Date()
    }

    /// Check if we recently authenticated (within last 5 seconds)
    static var recentlyAuthenticated: Bool {
        guard let lastAuth = lastAuthenticatedAt else { return false }
        return Date().timeIntervalSince(lastAuth) < 5.0
    }

    /// Clear all authentication data (used for sign out)
    static func clearAuthentication() {
        apiKey = ""
        deviceId = nil
        userEmail = nil
        lastAuthenticatedAt = nil
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
    static let authenticationRequired = Notification.Name("authenticationRequired")
}
