import Foundation
import SwiftUI

/// Service for handling push notifications and auth requests
actor NotificationService {
    static let shared = NotificationService()

    private var fcmToken: String?
    private var pendingAuthRequest: AuthRequest?

    // Published state for UI binding
    @MainActor @Published var currentAuthRequest: AuthRequest?
    @MainActor @Published var showAuthRequestSheet = false

    private init() {}

    // MARK: - FCM Token Management

    func updateFCMToken(_ token: String) async {
        self.fcmToken = token
        await Logger.shared.info("FCM token updated")

        // Register device with server if configured
        if AppSettings.isConfigured {
            await registerDeviceWithServer()
        }
    }

    func getFCMToken() -> String? {
        return fcmToken
    }

    // MARK: - Device Registration

    func registerDeviceWithServer() async {
        guard let token = fcmToken else {
            await Logger.shared.warning("Cannot register device: no FCM token")
            return
        }

        do {
            let response = try await APIService.shared.registerDevice(
                fcmToken: token,
                name: await getDeviceName()
            )
            await Logger.shared.info("Device registered with server: \(response.id)")

            // Save device ID
            await MainActor.run {
                AppSettings.deviceId = response.id
            }
        } catch {
            await Logger.shared.error("Failed to register device: \(error)")
        }
    }

    private func getDeviceName() async -> String {
        await MainActor.run {
            UIDevice.current.name
        }
    }

    // MARK: - Auth Request Handling

    func handleAuthRequest(id: String, userInfo: [AnyHashable: Any]) async {
        let request = AuthRequest(
            id: id,
            email: userInfo["email"] as? String ?? "Unknown",
            ipAddress: userInfo["ipAddress"] as? String,
            userAgent: userInfo["userAgent"] as? String,
            timestamp: Date()
        )

        self.pendingAuthRequest = request

        await MainActor.run {
            self.currentAuthRequest = request
        }
    }

    func showAuthRequestUI(id: String, userInfo: [AnyHashable: Any]) async {
        await handleAuthRequest(id: id, userInfo: userInfo)

        if let request = pendingAuthRequest {
            await MainActor.run {
                self.showAuthRequestSheet = true
                NotificationCenter.default.post(name: .showAuthRequest, object: request)
            }
        }
    }

    func approveAuthRequest() async {
        guard let request = pendingAuthRequest else {
            await Logger.shared.warning("No pending auth request to approve")
            return
        }

        do {
            try await APIService.shared.respondToAuthRequest(
                id: request.id,
                approved: true
            )
            await Logger.shared.info("Auth request approved: \(request.id)")

            await clearAuthRequest()
        } catch {
            await Logger.shared.error("Failed to approve auth request: \(error)")
        }
    }

    func denyAuthRequest() async {
        guard let request = pendingAuthRequest else {
            await Logger.shared.warning("No pending auth request to deny")
            return
        }

        do {
            try await APIService.shared.respondToAuthRequest(
                id: request.id,
                approved: false
            )
            await Logger.shared.info("Auth request denied: \(request.id)")

            await clearAuthRequest()
        } catch {
            await Logger.shared.error("Failed to deny auth request: \(error)")
        }
    }

    private func clearAuthRequest() async {
        pendingAuthRequest = nil

        await MainActor.run {
            self.currentAuthRequest = nil
            self.showAuthRequestSheet = false
        }
    }
}

// MARK: - Auth Request Model

struct AuthRequest: Identifiable {
    let id: String
    let email: String
    let ipAddress: String?
    let userAgent: String?
    let timestamp: Date

    var formattedTimestamp: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        return formatter.string(from: timestamp)
    }

    var browserInfo: String {
        guard let userAgent = userAgent else { return "Unknown browser" }

        // Parse user agent for common browsers
        if userAgent.contains("Chrome") {
            return "Chrome"
        } else if userAgent.contains("Safari") && !userAgent.contains("Chrome") {
            return "Safari"
        } else if userAgent.contains("Firefox") {
            return "Firefox"
        } else if userAgent.contains("Edge") {
            return "Edge"
        }
        return "Web browser"
    }
}
