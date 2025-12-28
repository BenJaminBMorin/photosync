import Foundation

/// App settings stored in UserDefaults
struct AppSettings {
    private static let serverURLKey = "serverURL"
    private static let apiKeyKey = "apiKey"
    private static let wifiOnlyKey = "wifiOnly"

    static var serverURL: String {
        get { UserDefaults.standard.string(forKey: serverURLKey) ?? "" }
        set { UserDefaults.standard.set(newValue, forKey: serverURLKey) }
    }

    static var apiKey: String {
        get { UserDefaults.standard.string(forKey: apiKeyKey) ?? "" }
        set { UserDefaults.standard.set(newValue, forKey: apiKeyKey) }
    }

    static var wifiOnly: Bool {
        get { UserDefaults.standard.bool(forKey: wifiOnlyKey) }
        set { UserDefaults.standard.set(newValue, forKey: wifiOnlyKey) }
    }

    static var isConfigured: Bool {
        !serverURL.isEmpty && !apiKey.isEmpty
    }

    /// Normalized server URL (removes trailing slash)
    static var normalizedServerURL: String {
        var url = serverURL.trimmingCharacters(in: .whitespacesAndNewlines)
        if url.hasSuffix("/") {
            url.removeLast()
        }
        return url
    }
}
