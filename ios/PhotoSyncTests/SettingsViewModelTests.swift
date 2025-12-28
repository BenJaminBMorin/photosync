import XCTest
@testable import PhotoSync

final class SettingsViewModelTests: XCTestCase {

    override func setUp() {
        super.setUp()
        // Clear UserDefaults before each test
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

    func testIsConfiguredReturnsFalseWhenEmpty() {
        let viewModel = SettingsViewModel()

        XCTAssertFalse(viewModel.isConfigured)
    }

    func testIsConfiguredReturnsFalseWithOnlyServerURL() {
        UserDefaults.standard.set("https://example.com", forKey: "serverURL")
        let viewModel = SettingsViewModel()

        XCTAssertFalse(viewModel.isConfigured)
    }

    func testIsConfiguredReturnsFalseWithOnlyAPIKey() {
        UserDefaults.standard.set("test-api-key", forKey: "apiKey")
        let viewModel = SettingsViewModel()

        XCTAssertFalse(viewModel.isConfigured)
    }

    func testIsConfiguredReturnsTrueWithBothValues() {
        UserDefaults.standard.set("https://example.com", forKey: "serverURL")
        UserDefaults.standard.set("test-api-key", forKey: "apiKey")
        let viewModel = SettingsViewModel()

        XCTAssertTrue(viewModel.isConfigured)
    }

    func testWifiOnlyDefaultsToTrue() {
        let viewModel = SettingsViewModel()

        XCTAssertTrue(viewModel.wifiOnly)
    }

    func testServerURLPersistsToUserDefaults() {
        let viewModel = SettingsViewModel()
        viewModel.serverURL = "https://test.example.com"

        XCTAssertEqual(UserDefaults.standard.string(forKey: "serverURL"), "https://test.example.com")
    }

    func testAPIKeyPersistsToUserDefaults() {
        let viewModel = SettingsViewModel()
        viewModel.apiKey = "my-secret-key"

        XCTAssertEqual(UserDefaults.standard.string(forKey: "apiKey"), "my-secret-key")
    }

    func testWifiOnlyPersistsToUserDefaults() {
        UserDefaults.standard.set(true, forKey: "wifiOnly") // Set initial value
        let viewModel = SettingsViewModel()
        viewModel.wifiOnly = false

        XCTAssertFalse(UserDefaults.standard.bool(forKey: "wifiOnly"))
    }

    func testTestResultInitiallyNil() {
        let viewModel = SettingsViewModel()

        XCTAssertNil(viewModel.testResult)
    }

    func testIsTestingInitiallyFalse() {
        let viewModel = SettingsViewModel()

        XCTAssertFalse(viewModel.isTesting)
    }
}
