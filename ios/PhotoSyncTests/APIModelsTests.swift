import XCTest
@testable import PhotoSync

final class APIModelsTests: XCTestCase {

    // MARK: - UploadResponse Tests

    func testUploadResponseDecoding() throws {
        let json = """
        {
            "id": "abc123",
            "storedPath": "2024/01/IMG_1234.jpg",
            "uploadedAt": "2024-01-15T10:30:00Z",
            "isDuplicate": false
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(UploadResponse.self, from: json)

        XCTAssertEqual(response.id, "abc123")
        XCTAssertEqual(response.storedPath, "2024/01/IMG_1234.jpg")
        XCTAssertEqual(response.uploadedAt, "2024-01-15T10:30:00Z")
        XCTAssertFalse(response.isDuplicate)
    }

    func testUploadResponseDecodingWithDuplicate() throws {
        let json = """
        {
            "id": "abc123",
            "storedPath": "2024/01/IMG_1234.jpg",
            "uploadedAt": "2024-01-15T10:30:00Z",
            "isDuplicate": true
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(UploadResponse.self, from: json)

        XCTAssertTrue(response.isDuplicate)
    }

    // MARK: - CheckHashesRequest Tests

    func testCheckHashesRequestEncoding() throws {
        let request = CheckHashesRequest(hashes: ["hash1", "hash2", "hash3"])

        let data = try JSONEncoder().encode(request)
        let json = try JSONSerialization.jsonObject(with: data) as! [String: Any]

        XCTAssertEqual(json["hashes"] as? [String], ["hash1", "hash2", "hash3"])
    }

    // MARK: - CheckHashesResponse Tests

    func testCheckHashesResponseDecoding() throws {
        let json = """
        {
            "existing": ["hash1", "hash3"],
            "missing": ["hash2"]
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(CheckHashesResponse.self, from: json)

        XCTAssertEqual(response.existing, ["hash1", "hash3"])
        XCTAssertEqual(response.missing, ["hash2"])
    }

    // MARK: - PhotoResponse Tests

    func testPhotoResponseDecoding() throws {
        let json = """
        {
            "id": "photo123",
            "originalFilename": "IMG_1234.jpg",
            "storedPath": "2024/01/IMG_1234.jpg",
            "fileSize": 2048576,
            "dateTaken": "2024-01-15T10:30:00Z",
            "uploadedAt": "2024-01-16T12:00:00Z"
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(PhotoResponse.self, from: json)

        XCTAssertEqual(response.id, "photo123")
        XCTAssertEqual(response.originalFilename, "IMG_1234.jpg")
        XCTAssertEqual(response.storedPath, "2024/01/IMG_1234.jpg")
        XCTAssertEqual(response.fileSize, 2048576)
        XCTAssertEqual(response.dateTaken, "2024-01-15T10:30:00Z")
        XCTAssertEqual(response.uploadedAt, "2024-01-16T12:00:00Z")
    }

    // MARK: - PhotoListResponse Tests

    func testPhotoListResponseDecoding() throws {
        let json = """
        {
            "photos": [
                {
                    "id": "photo1",
                    "originalFilename": "IMG_001.jpg",
                    "storedPath": "2024/01/IMG_001.jpg",
                    "fileSize": 1024,
                    "dateTaken": "2024-01-15T10:00:00Z",
                    "uploadedAt": "2024-01-16T12:00:00Z"
                }
            ],
            "totalCount": 100,
            "skip": 0,
            "take": 20
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(PhotoListResponse.self, from: json)

        XCTAssertEqual(response.photos.count, 1)
        XCTAssertEqual(response.totalCount, 100)
        XCTAssertEqual(response.skip, 0)
        XCTAssertEqual(response.take, 20)
    }

    // MARK: - HealthResponse Tests

    func testHealthResponseDecoding() throws {
        let json = """
        {
            "status": "healthy",
            "timestamp": "2024-01-15T10:30:00Z"
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(HealthResponse.self, from: json)

        XCTAssertEqual(response.status, "healthy")
        XCTAssertEqual(response.timestamp, "2024-01-15T10:30:00Z")
    }

    // MARK: - ErrorResponse Tests

    func testErrorResponseDecoding() throws {
        let json = """
        {
            "error": "Invalid API key"
        }
        """.data(using: .utf8)!

        let response = try JSONDecoder().decode(ErrorResponse.self, from: json)

        XCTAssertEqual(response.error, "Invalid API key")
    }
}
