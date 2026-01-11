import Foundation
import Security

/// Service for securely storing sensitive data in the iOS Keychain
enum KeychainService {
    private static let serviceName = "com.photosync.app"

    enum KeychainError: Error {
        case duplicateItem
        case itemNotFound
        case unexpectedStatus(OSStatus)
        case encodingError
    }

    // MARK: - API Key Storage

    private static let apiKeyAccount = "apiKey"

    /// Store the API key securely in the Keychain
    static func setAPIKey(_ apiKey: String) throws {
        guard let data = apiKey.data(using: .utf8) else {
            throw KeychainError.encodingError
        }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: apiKeyAccount,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        // First try to delete any existing item
        SecItemDelete(query as CFDictionary)

        // Add the new item
        let status = SecItemAdd(query as CFDictionary, nil)

        guard status == errSecSuccess else {
            throw KeychainError.unexpectedStatus(status)
        }
    }

    /// Retrieve the API key from the Keychain
    static func getAPIKey() -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: apiKeyAccount,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let data = result as? Data,
              let apiKey = String(data: data, encoding: .utf8) else {
            return nil
        }

        return apiKey
    }

    /// Delete the API key from the Keychain
    static func deleteAPIKey() {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: apiKeyAccount
        ]

        SecItemDelete(query as CFDictionary)
    }

    /// Check if API key exists in Keychain
    static func hasAPIKey() -> Bool {
        return getAPIKey() != nil
    }

    // MARK: - Login Password Storage

    private static let loginPasswordAccount = "loginPassword"

    /// Store the login password securely in the Keychain
    static func setLoginPassword(_ password: String) throws {
        guard let data = password.data(using: .utf8) else {
            throw KeychainError.encodingError
        }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: loginPasswordAccount,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        // First try to delete any existing item
        SecItemDelete(query as CFDictionary)

        // Add the new item
        let status = SecItemAdd(query as CFDictionary, nil)

        guard status == errSecSuccess else {
            throw KeychainError.unexpectedStatus(status)
        }
    }

    /// Retrieve the login password from the Keychain
    static func getLoginPassword() -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: loginPasswordAccount,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let data = result as? Data,
              let password = String(data: data, encoding: .utf8) else {
            return nil
        }

        return password
    }

    /// Delete the login password from the Keychain
    static func deleteLoginPassword() {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: loginPasswordAccount
        ]

        SecItemDelete(query as CFDictionary)
    }

    // MARK: - Migration

    /// Migrate API key from UserDefaults to Keychain (one-time migration)
    static func migrateFromUserDefaults() {
        let userDefaultsKey = "apiKey"

        // Check if already migrated (key exists in Keychain)
        if hasAPIKey() {
            // Clean up UserDefaults after successful migration
            UserDefaults.standard.removeObject(forKey: userDefaultsKey)
            return
        }

        // Get API key from UserDefaults
        guard let oldAPIKey = UserDefaults.standard.string(forKey: userDefaultsKey),
              !oldAPIKey.isEmpty else {
            return
        }

        // Store in Keychain
        do {
            try setAPIKey(oldAPIKey)
            // Remove from UserDefaults after successful migration
            UserDefaults.standard.removeObject(forKey: userDefaultsKey)
        } catch {
            // Migration failed, keep in UserDefaults for now
            print("Failed to migrate API key to Keychain: \(error)")
        }
    }
}
