import Foundation

/// Service for communicating with the PhotoSync server API
actor APIService {
    static let shared = APIService()

    private let session: URLSession

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 120 // Longer for uploads
        self.session = URLSession(configuration: config)
    }

    // MARK: - Health Check

    /// Test connection to the server
    func healthCheck() async throws -> HealthResponse {
        let url = try buildURL(path: "/api/health")
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        // Health check doesn't require API key

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return try JSONDecoder().decode(HealthResponse.self, from: data)
    }

    // MARK: - Photo Upload

    /// Upload a single photo
    func uploadPhoto(
        imageData: Data,
        filename: String,
        dateTaken: Date,
        deviceId: String? = nil
    ) async throws -> UploadResponse {
        let url = try buildURL(path: "/api/photos/upload")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        addAPIKeyHeader(to: &request)

        let boundary = UUID().uuidString
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        var body = Data()

        // Add file
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"file\"; filename=\"\(filename)\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: image/jpeg\r\n\r\n".data(using: .utf8)!)
        body.append(imageData)
        body.append("\r\n".data(using: .utf8)!)

        // Add originalFilename
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"originalFilename\"\r\n\r\n".data(using: .utf8)!)
        body.append("\(filename)\r\n".data(using: .utf8)!)

        // Add dateTaken
        let dateFormatter = ISO8601DateFormatter()
        let dateString = dateFormatter.string(from: dateTaken)
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"dateTaken\"\r\n\r\n".data(using: .utf8)!)
        body.append("\(dateString)\r\n".data(using: .utf8)!)

        // Add deviceId if provided
        if let deviceId = deviceId {
            body.append("--\(boundary)\r\n".data(using: .utf8)!)
            body.append("Content-Disposition: form-data; name=\"deviceId\"\r\n\r\n".data(using: .utf8)!)
            body.append("\(deviceId)\r\n".data(using: .utf8)!)
        }

        body.append("--\(boundary)--\r\n".data(using: .utf8)!)

        request.httpBody = body

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return try JSONDecoder().decode(UploadResponse.self, from: data)
    }

    // MARK: - Check Hashes

    /// Check which hashes already exist on the server
    func checkHashes(_ hashes: [String]) async throws -> CheckHashesResponse {
        let url = try buildURL(path: "/api/photos/check")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &request)

        let body = CheckHashesRequest(hashes: hashes)
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return try JSONDecoder().decode(CheckHashesResponse.self, from: data)
    }

    // MARK: - List Photos

    /// Get list of photos on server
    func listPhotos(skip: Int = 0, take: Int = 50) async throws -> PhotoListResponse {
        var components = URLComponents(string: AppSettings.normalizedServerURL + "/api/photos")!
        components.queryItems = [
            URLQueryItem(name: "skip", value: String(skip)),
            URLQueryItem(name: "take", value: String(take))
        ]

        guard let url = components.url else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        addAPIKeyHeader(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return try JSONDecoder().decode(PhotoListResponse.self, from: data)
    }

    // MARK: - Download Photo

    /// Download photo thumbnail from server
    func downloadThumbnail(photoId: String) async throws -> Data {
        let url = try buildURL(path: "/api/photos/\(photoId)/thumbnail")
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        addAPIKeyHeader(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return data
    }

    /// Download full photo from server
    func downloadPhoto(photoId: String) async throws -> Data {
        let url = try buildURL(path: "/api/photos/\(photoId)")
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        addAPIKeyHeader(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return data
    }

    /// Delete photo from server
    func deletePhoto(photoId: String) async throws {
        let url = try buildURL(path: "/api/photos/\(photoId)")
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        addAPIKeyHeader(to: &request)

        let (_, response) = try await session.data(for: request)
        try validateResponse(response)
    }

    /// Delete multiple photos from server
    func deletePhotos(photoIds: [String]) async throws {
        let url = try buildURL(path: "/api/photos/batch-delete")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &request)

        let body = ["photoIds": photoIds]
        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await session.data(for: request)
        try validateResponse(response)
    }

    // MARK: - Device Registration

    /// Register device for push notifications
    func registerDevice(fcmToken: String, name: String) async throws -> DeviceResponse {
        let url = try buildURL(path: "/api/devices/register")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &request)

        let body = RegisterDeviceRequest(fcmToken: fcmToken, deviceName: name, platform: "ios")
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        return try JSONDecoder().decode(DeviceResponse.self, from: data)
    }

    // MARK: - Auth Response

    /// Respond to a web authentication request
    func respondToAuthRequest(id: String, approved: Bool) async throws {
        await Logger.shared.info("APIService.respondToAuthRequest called - id: \(id), approved: \(approved)")

        let url = try buildURL(path: "/api/web/auth/respond")
        await Logger.shared.info("API URL: \(url.absoluteString)")

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &request)

        let deviceId = AppSettings.deviceId
        let body = AuthResponseRequest(requestId: id, approved: approved, deviceId: deviceId)
        request.httpBody = try JSONEncoder().encode(body)

        await Logger.shared.info("Sending POST request to /api/web/auth/respond with deviceId: \(deviceId ?? "nil")")

        let (data, response) = try await session.data(for: request)

        await Logger.shared.info("Response received - status: \((response as? HTTPURLResponse)?.statusCode ?? -1)")

        try validateResponse(response)
        await Logger.shared.info("Response validation passed")
    }

    // MARK: - Delete Response

    /// Respond to a photo deletion request
    func respondToDeleteRequest(id: String, approved: Bool) async throws {
        await Logger.shared.info("APIService.respondToDeleteRequest called - id: \(id), approved: \(approved)")

        let url = try buildURL(path: "/api/web/delete/respond")
        await Logger.shared.info("API URL: \(url.absoluteString)")

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &request)

        let deviceId = AppSettings.deviceId
        let body = DeleteResponseRequest(requestId: id, approved: approved, deviceId: deviceId)
        request.httpBody = try JSONEncoder().encode(body)

        await Logger.shared.info("Sending POST request to /api/web/delete/respond with deviceId: \(deviceId ?? "nil")")

        let (data, response) = try await session.data(for: request)

        await Logger.shared.info("Response received - status: \((response as? HTTPURLResponse)?.statusCode ?? -1)")

        try validateResponse(response)
        await Logger.shared.info("Response validation passed")
    }

    // MARK: - Helpers

    private func buildURL(path: String) throws -> URL {
        let baseURL = AppSettings.normalizedServerURL
        guard !baseURL.isEmpty, let url = URL(string: baseURL + path) else {
            throw APIError.invalidURL
        }
        return url
    }

    // MARK: - Sync Endpoints

    func getSyncStatus(deviceId: String?) async throws -> SyncStatusResponse {
        await Logger.shared.info("API: getSyncStatus called with deviceId: \(deviceId ?? "nil")")
        let url = try buildURL(path: "/api/sync/status")
        var request = URLRequest(url: url)
        addAPIKeyHeader(to: &request)

        if let deviceId = deviceId {
            request.setValue(deviceId, forHTTPHeaderField: "X-Device-ID")
        }

        await Logger.shared.info("API: Making request to: \(url.absoluteString)")
        let (data, response) = try await session.data(for: request)

        // Log response for debugging
        if let httpResponse = response as? HTTPURLResponse {
            await Logger.shared.info("API: Response status: \(httpResponse.statusCode)")
            if httpResponse.statusCode != 200 {
                let responseBody = String(data: data, encoding: .utf8) ?? "Unable to decode response"
                await Logger.shared.error("API: Error response body: \(responseBody)")
            }
        }

        try validateResponse(response)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(SyncStatusResponse.self, from: data)
    }

    func syncPhotos(request: SyncPhotosRequest) async throws -> SyncPhotosResponse {
        let url = try buildURL(path: "/api/sync/photos")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        addAPIKeyHeader(to: &urlRequest)
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        urlRequest.httpBody = try encoder.encode(request)

        let (data, response) = try await session.data(for: urlRequest)
        try validateResponse(response)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(SyncPhotosResponse.self, from: data)
    }

    func getLegacyPhotos(limit: Int = 100) async throws -> LegacyPhotosResponse {
        let baseUrl = try buildURL(path: "/api/sync/legacy-photos")
        var components = URLComponents(url: baseUrl, resolvingAgainstBaseURL: false)!
        components.queryItems = [URLQueryItem(name: "limit", value: String(limit))]

        var request = URLRequest(url: components.url!)
        addAPIKeyHeader(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(LegacyPhotosResponse.self, from: data)
    }

    func claimLegacyPhotos(deviceId: String, claimAll: Bool = true, photoIds: [String]? = nil) async throws -> ClaimLegacyResponse {
        let url = try buildURL(path: "/api/sync/claim-legacy")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        addAPIKeyHeader(to: &urlRequest)
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let request = ClaimLegacyRequest(deviceId: deviceId, claimAll: claimAll, photoIds: photoIds)
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (data, response) = try await session.data(for: urlRequest)
        try validateResponse(response)

        return try JSONDecoder().decode(ClaimLegacyResponse.self, from: data)
    }

    // MARK: - Collections

    /// Get all collections from server
    func getCollections() async throws -> CollectionsListResponse {
        let url = try buildURL(path: "/api/collections")
        var request = URLRequest(url: url)
        addAPIKeyHeader(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(CollectionsListResponse.self, from: data)
    }

    /// Create a new collection
    func createCollection(name: String) async throws -> CollectionResponse {
        let url = try buildURL(path: "/api/collections")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &urlRequest)

        let request = CreateCollectionRequest(name: name)
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (data, response) = try await session.data(for: urlRequest)
        try validateResponse(response)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(CollectionResponse.self, from: data)
    }

    /// Delete a collection
    func deleteCollection(collectionId: String) async throws {
        let url = try buildURL(path: "/api/collections/\(collectionId)")
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        addAPIKeyHeader(to: &request)

        let (_, response) = try await session.data(for: request)
        try validateResponse(response)
    }

    /// Add photos to a collection
    func addPhotosToCollection(collectionId: String, photoIds: [String]) async throws {
        let url = try buildURL(path: "/api/collections/\(collectionId)/photos")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &urlRequest)

        let request = ManageCollectionPhotosRequest(photoIds: photoIds)
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (_, response) = try await session.data(for: urlRequest)
        try validateResponse(response)
    }

    /// Remove photos from a collection
    func removePhotosFromCollection(collectionId: String, photoIds: [String]) async throws {
        let url = try buildURL(path: "/api/collections/\(collectionId)/photos")
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "DELETE"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAPIKeyHeader(to: &urlRequest)

        let request = ManageCollectionPhotosRequest(photoIds: photoIds)
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (_, response) = try await session.data(for: urlRequest)
        try validateResponse(response)
    }

    // MARK: - Helper Methods

    private func addAPIKeyHeader(to request: inout URLRequest) {
        let apiKey = AppSettings.apiKey
        if !apiKey.isEmpty {
            request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
            // Debug logging (mask most of the key for security)
            let maskedKey = apiKey.count > 8
                ? "\(apiKey.prefix(4))...\(apiKey.suffix(4))"
                : "****"
            Task {
                await Logger.shared.info("API: Adding API key header: X-API-Key = \(maskedKey)")
            }
        } else {
            Task {
                await Logger.shared.error("API: WARNING - API key is empty, header not added")
            }
        }
    }

    private func validateResponse(_ response: URLResponse) throws {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        switch httpResponse.statusCode {
        case 200...299:
            return
        case 401:
            throw APIError.unauthorized
        case 400:
            throw APIError.badRequest
        case 404:
            throw APIError.notFound
        default:
            throw APIError.serverError(httpResponse.statusCode)
        }
    }
}

enum APIError: Error, LocalizedError {
    case invalidURL
    case invalidResponse
    case unauthorized
    case badRequest
    case notFound
    case serverError(Int)
    case networkError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid server URL"
        case .invalidResponse:
            return "Invalid response from server"
        case .unauthorized:
            return "Invalid API key"
        case .badRequest:
            return "Bad request"
        case .notFound:
            return "Resource not found"
        case .serverError(let code):
            return "Server error (\(code))"
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        }
    }
}
