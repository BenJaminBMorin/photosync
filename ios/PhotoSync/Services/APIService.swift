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
        dateTaken: Date
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

        let body = AuthResponseRequest(requestId: id, approved: approved)
        request.httpBody = try JSONEncoder().encode(body)

        await Logger.shared.info("Sending POST request to /api/web/auth/respond")

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

    private func addAPIKeyHeader(to request: inout URLRequest) {
        let apiKey = AppSettings.apiKey
        if !apiKey.isEmpty {
            request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
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
