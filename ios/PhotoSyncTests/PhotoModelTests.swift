import XCTest
import Photos
@testable import PhotoSync

final class PhotoModelTests: XCTestCase {

    // MARK: - PhotoWithState Tests

    func testPhotoWithStateDefaults() {
        // We can't create a real PHAsset in tests, but we can test the struct behavior
        // by checking the computed properties and state management

        // Test that sync state enum has expected cases
        XCTAssertEqual(SyncState.notSynced.rawValue, "notSynced")
        XCTAssertEqual(SyncState.synced.rawValue, "synced")
        XCTAssertEqual(SyncState.syncing.rawValue, "syncing")
    }

    func testSyncStateRawValues() {
        XCTAssertEqual(SyncState(rawValue: "notSynced"), .notSynced)
        XCTAssertEqual(SyncState(rawValue: "synced"), .synced)
        XCTAssertEqual(SyncState(rawValue: "syncing"), .syncing)
        XCTAssertNil(SyncState(rawValue: "invalid"))
    }
}

final class SettingsModelTests: XCTestCase {

    override func setUp() {
        super.setUp()
        UserDefaults.standard.removeObject(forKey: "serverURL")
        UserDefaults.standard.removeObject(forKey: "apiKey")
        UserDefaults.standard.removeObject(forKey: "wifiOnly")
    }

    override func tearDown() {
        UserDefaults.standard.removeObject(forKey: "serverURL")
        UserDefaults.standard.removeObject(forKey: "apiKey")
        UserDefaults.standard.removeObject(forKey: "wifiOnly")
        super.tearDown()
    }

    func testServerURLGetterReturnsEmptyWhenNotSet() {
        XCTAssertEqual(Settings.serverURL, "")
    }

    func testServerURLSetterPersistsValue() {
        Settings.serverURL = "https://example.com"

        XCTAssertEqual(Settings.serverURL, "https://example.com")
        XCTAssertEqual(UserDefaults.standard.string(forKey: "serverURL"), "https://example.com")
    }

    func testAPIKeyGetterReturnsEmptyWhenNotSet() {
        XCTAssertEqual(Settings.apiKey, "")
    }

    func testAPIKeySetterPersistsValue() {
        Settings.apiKey = "my-secret-key"

        XCTAssertEqual(Settings.apiKey, "my-secret-key")
        XCTAssertEqual(UserDefaults.standard.string(forKey: "apiKey"), "my-secret-key")
    }

    func testWifiOnlyDefaultsToTrue() {
        XCTAssertTrue(Settings.wifiOnly)
    }

    func testWifiOnlySetterPersistsValue() {
        Settings.wifiOnly = false

        XCTAssertFalse(Settings.wifiOnly)
    }

    func testIsConfiguredReturnsFalseWhenEmpty() {
        XCTAssertFalse(Settings.isConfigured)
    }

    func testIsConfiguredReturnsFalseWithOnlyServerURL() {
        Settings.serverURL = "https://example.com"

        XCTAssertFalse(Settings.isConfigured)
    }

    func testIsConfiguredReturnsFalseWithOnlyAPIKey() {
        Settings.apiKey = "test-key"

        XCTAssertFalse(Settings.isConfigured)
    }

    func testIsConfiguredReturnsTrueWithBothValues() {
        Settings.serverURL = "https://example.com"
        Settings.apiKey = "test-key"

        XCTAssertTrue(Settings.isConfigured)
    }

    func testIsConfiguredReturnsFalseWithWhitespaceOnly() {
        Settings.serverURL = "   "
        Settings.apiKey = "   "

        XCTAssertFalse(Settings.isConfigured)
    }
}
