import Foundation
import UIKit

/// Service for handling invite token redemption
actor InviteService {
    static let shared = InviteService()

    private let session: URLSession

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: config)
    }

    /// Redeem an invite token and configure the app
    func redeemInvite(token: String) async throws -> RedeemInviteResponse {
        await Logger.shared.info("Starting invite redemption for token")

        // Decode the base64 token to extract server URL and random token
        guard let decodedData = Data(base64Encoded: token, options: .ignoreUnknownCharacters),
              let decodedString = String(data: decodedData, encoding: .utf8) else {
            await Logger.shared.error("Failed to decode base64 token")
            throw InviteError.invalidToken
        }

        await Logger.shared.info("Token decoded successfully: \(decodedString.prefix(50))...")

        // Split on "|" to get random token and server URL
        let parts = decodedString.split(separator: "|", maxSplits: 1)
        guard parts.count == 2 else {
            await Logger.shared.error("Invalid token structure - expected format: random|serverURL")
            throw InviteError.invalidToken
        }

        let randomToken = String(parts[0])
        let serverURL = String(parts[1])

        await Logger.shared.info("Extracted server URL: \(serverURL)")

        // Get device info
        let deviceInfo = await getDeviceInfo()

        // Build the API URL using the extracted server URL
        let normalizedServerURL = serverURL.hasSuffix("/") ? String(serverURL.dropLast()) : serverURL
        guard let url = URL(string: "\(normalizedServerURL)/api/invite/redeem") else {
            await Logger.shared.error("Invalid server URL: \(serverURL)")
            throw InviteError.invalidServerURL
        }

        await Logger.shared.info("API URL: \(url.absoluteString)")

        // Create request
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        // Create request body with the original encoded token
        let body = RedeemInviteRequest(token: token, deviceInfo: deviceInfo)
        request.httpBody = try JSONEncoder().encode(body)

        await Logger.shared.info("Sending redemption request...")

        // Make the request
        let (data, response) = try await session.data(for: request)

        // Validate response
        guard let httpResponse = response as? HTTPURLResponse else {
            await Logger.shared.error("Invalid response type")
            throw InviteError.invalidResponse
        }

        await Logger.shared.info("Response status: \(httpResponse.statusCode)")

        switch httpResponse.statusCode {
        case 200...299:
            // Success - decode the response
            let redeemResponse = try JSONDecoder().decode(RedeemInviteResponse.self, from: data)
            await Logger.shared.info("Successfully redeemed invite for user: \(redeemResponse.email)")

            // Save credentials to AppSettings and Keychain
            await saveCredentials(response: redeemResponse)

            return redeemResponse

        case 404:
            await Logger.shared.error("Invite token not found")
            throw InviteError.tokenNotFound

        case 410:
            await Logger.shared.error("Invite token already used or expired")
            throw InviteError.tokenExpiredOrUsed

        case 400:
            await Logger.shared.error("Invalid token format")
            throw InviteError.invalidToken

        default:
            await Logger.shared.error("Server error: \(httpResponse.statusCode)")
            throw InviteError.serverError(httpResponse.statusCode)
        }
    }

    /// Save the credentials to AppSettings and Keychain
    private func saveCredentials(response: RedeemInviteResponse) async {
        await Logger.shared.info("Saving credentials for: \(response.email)")

        // Save to AppSettings (synchronously on main actor)
        await MainActor.run {
            // Normalize server URL
            let normalizedURL = response.serverUrl.hasSuffix("/")
                ? String(response.serverUrl.dropLast())
                : response.serverUrl

            AppSettings.serverURL = normalizedURL

            // Save API key to Keychain
            KeychainService.saveAPIKey(response.apiKey)

            await Logger.shared.info("Credentials saved successfully")
            await Logger.shared.info("Server URL: \(normalizedURL)")
            await Logger.shared.info("User ID: \(response.userId)")
        }
    }

    /// Get device info for tracking
    private func getDeviceInfo() async -> String {
        await MainActor.run {
            let device = UIDevice.current
            let model = device.model
            let systemVersion = device.systemVersion
            let deviceName = device.name

            return "\(deviceName) - \(model) iOS \(systemVersion)"
        }
    }
}

enum InviteError: Error, LocalizedError {
    case invalidToken
    case invalidServerURL
    case invalidResponse
    case tokenNotFound
    case tokenExpiredOrUsed
    case serverError(Int)
    case networkError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidToken:
            return "Invalid or malformed invite token"
        case .invalidServerURL:
            return "Invalid server URL in invite token"
        case .invalidResponse:
            return "Invalid response from server"
        case .tokenNotFound:
            return "Invite token not found"
        case .tokenExpiredOrUsed:
            return "This invite has already been used or has expired"
        case .serverError(let code):
            return "Server error (\(code))"
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        }
    }
}
