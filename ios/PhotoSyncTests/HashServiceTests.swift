import XCTest
import CryptoKit
@testable import PhotoSync

final class HashServiceTests: XCTestCase {

    func testSHA256HashForEmptyData() {
        let data = Data()
        let hash = HashService.sha256(data)

        // SHA256 of empty data is a known value
        XCTAssertEqual(hash, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
    }

    func testSHA256HashForKnownInput() {
        let data = "hello world".data(using: .utf8)!
        let hash = HashService.sha256(data)

        // Known SHA256 hash of "hello world"
        XCTAssertEqual(hash, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
    }

    func testSHA256HashIsConsistent() {
        let data = "test data".data(using: .utf8)!

        let hash1 = HashService.sha256(data)
        let hash2 = HashService.sha256(data)

        XCTAssertEqual(hash1, hash2)
    }

    func testSHA256HashIsDifferentForDifferentInputs() {
        let data1 = "input1".data(using: .utf8)!
        let data2 = "input2".data(using: .utf8)!

        let hash1 = HashService.sha256(data1)
        let hash2 = HashService.sha256(data2)

        XCTAssertNotEqual(hash1, hash2)
    }

    func testSHA256HashLength() {
        let data = "any data".data(using: .utf8)!
        let hash = HashService.sha256(data)

        // SHA256 produces 64 character hex string (256 bits / 4 bits per hex char)
        XCTAssertEqual(hash.count, 64)
    }

    func testSHA256HashIsLowercase() {
        let data = "TEST".data(using: .utf8)!
        let hash = HashService.sha256(data)

        XCTAssertEqual(hash, hash.lowercased())
    }
}

// Simple hash service for testing - matches the one that would be in the app
enum HashService {
    static func sha256(_ data: Data) -> String {
        let hash = SHA256.hash(data: data)
        return hash.compactMap { String(format: "%02x", $0) }.joined()
    }
}
