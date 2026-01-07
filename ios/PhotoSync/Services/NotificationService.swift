import Foundation
import SwiftUI

/// Service for handling push notifications and auth requests
actor NotificationService {
    static let shared = NotificationService()

    private var fcmToken: String?
    private var pendingAuthRequest: AuthRequest?
    private var pendingDeleteRequest: DeleteRequest?

    // Published state for UI binding
    @MainActor @Published var currentAuthRequest: AuthRequest?
    @MainActor @Published var showAuthRequestSheet = false
    @MainActor @Published var currentDeleteRequest: DeleteRequest?
    @MainActor @Published var showDeleteRequestSheet = false

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

        // Auto-show UI
        await showAuthRequestUI(id: id, userInfo: userInfo)
    }

    func showAuthRequestUI(id: String, userInfo: [AnyHashable: Any]) async {
        await Logger.shared.info("showAuthRequestUI called with id: \(id)")
        await Logger.shared.info("userInfo keys: \(userInfo.keys)")
        await Logger.shared.info("userInfo: \(userInfo)")

        let request = AuthRequest(
            id: id,
            email: userInfo["email"] as? String ?? "Unknown",
            ipAddress: userInfo["ipAddress"] as? String,
            userAgent: userInfo["userAgent"] as? String,
            timestamp: Date()
        )

        await Logger.shared.info("Created AuthRequest - email: \(request.email), ip: \(request.ipAddress ?? "nil"), userAgent: \(request.userAgent ?? "nil")")

        self.pendingAuthRequest = request

        await MainActor.run {
            self.currentAuthRequest = request
            self.showAuthRequestSheet = true
            NotificationCenter.default.post(name: .showAuthRequest, object: request)
        }

        await Logger.shared.info("Sheet should now be visible with request data")
    }

    // MARK: - Delete Request Handling

    func handleDeleteRequest(id: String, userInfo: [AnyHashable: Any]) async {
        await Logger.shared.info("Delete request received: \(id)")

        // Parse photo IDs from comma-separated string
        let photoIdsString = userInfo["photoIds"] as? String ?? ""
        let photoIds = photoIdsString.split(separator: ",").map { String($0) }

        let request = DeleteRequest(
            id: id,
            photoIds: photoIds,
            email: userInfo["email"] as? String ?? "Unknown",
            ipAddress: userInfo["ipAddress"] as? String,
            userAgent: userInfo["userAgent"] as? String,
            timestamp: Date()
        )

        self.pendingDeleteRequest = request

        await MainActor.run {
            self.currentDeleteRequest = request
        }

        // Auto-show UI
        await showDeleteRequestUI(id: id, userInfo: userInfo)
    }

    func showDeleteRequestUI(id: String, userInfo: [AnyHashable: Any]) async {
        await Logger.shared.info("showDeleteRequestUI called with id: \(id)")
        await Logger.shared.info("userInfo: \(userInfo)")

        // Parse photo IDs from comma-separated string
        let photoIdsString = userInfo["photoIds"] as? String ?? ""
        let photoIds = photoIdsString.split(separator: ",").map { String($0) }

        let request = DeleteRequest(
            id: id,
            photoIds: photoIds,
            email: userInfo["email"] as? String ?? "Unknown",
            ipAddress: userInfo["ipAddress"] as? String,
            userAgent: userInfo["userAgent"] as? String,
            timestamp: Date()
        )

        await Logger.shared.info("Created DeleteRequest - email: \(request.email), photoCount: \(request.photoCount)")

        self.pendingDeleteRequest = request

        await MainActor.run {
            self.currentDeleteRequest = request
            self.showDeleteRequestSheet = true
            NotificationCenter.default.post(name: .showDeleteRequest, object: request)
        }

        await Logger.shared.info("Delete sheet should now be visible with request data")
    }

    func approveDeleteRequest() async {
        await Logger.shared.info("approveDeleteRequest() called")

        guard let request = pendingDeleteRequest else {
            await Logger.shared.warning("No pending delete request to approve")
            return
        }

        await Logger.shared.info("Sending approval for delete request: \(request.id)")

        do {
            try await APIService.shared.respondToDeleteRequest(
                id: request.id,
                approved: true
            )
            await Logger.shared.info("Delete request approved successfully: \(request.id)")

            await clearDeleteRequest()
            await Logger.shared.info("Delete request cleared")
        } catch {
            await Logger.shared.error("Failed to approve delete request: \(error)")
            await Logger.shared.error("Error details: \(String(describing: error))")
        }
    }

    func denyDeleteRequest() async {
        await Logger.shared.info("denyDeleteRequest() called")

        guard let request = pendingDeleteRequest else {
            await Logger.shared.warning("No pending delete request to deny")
            return
        }

        await Logger.shared.info("Sending denial for delete request: \(request.id)")

        do {
            try await APIService.shared.respondToDeleteRequest(
                id: request.id,
                approved: false
            )
            await Logger.shared.info("Delete request denied successfully: \(request.id)")

            await clearDeleteRequest()
            await Logger.shared.info("Delete request cleared")
        } catch {
            await Logger.shared.error("Failed to deny delete request: \(error)")
            await Logger.shared.error("Error details: \(String(describing: error))")
        }
    }

    private func clearDeleteRequest() async {
        pendingDeleteRequest = nil

        await MainActor.run {
            self.currentDeleteRequest = nil
            self.showDeleteRequestSheet = false
        }
    }

    func approveAuthRequest() async {
        await Logger.shared.info("approveAuthRequest() called")

        guard let request = pendingAuthRequest else {
            await Logger.shared.warning("No pending auth request to approve")
            return
        }

        await Logger.shared.info("Sending approval for request: \(request.id)")

        do {
            try await APIService.shared.respondToAuthRequest(
                id: request.id,
                approved: true
            )
            await Logger.shared.info("Auth request approved successfully: \(request.id)")

            await clearAuthRequest()
            await Logger.shared.info("Auth request cleared")
        } catch {
            await Logger.shared.error("Failed to approve auth request: \(error)")
            await Logger.shared.error("Error details: \(String(describing: error))")
        }
    }

    func denyAuthRequest() async {
        await Logger.shared.info("denyAuthRequest() called")

        guard let request = pendingAuthRequest else {
            await Logger.shared.warning("No pending auth request to deny")
            return
        }

        await Logger.shared.info("Sending denial for request: \(request.id)")

        do {
            try await APIService.shared.respondToAuthRequest(
                id: request.id,
                approved: false
            )
            await Logger.shared.info("Auth request denied successfully: \(request.id)")

            await clearAuthRequest()
            await Logger.shared.info("Auth request cleared")
        } catch {
            await Logger.shared.error("Failed to deny auth request: \(error)")
            await Logger.shared.error("Error details: \(String(describing: error))")
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

// MARK: - Delete Request Model

struct DeleteRequest: Identifiable {
    let id: String
    let photoIds: [String]
    let email: String
    let ipAddress: String?
    let userAgent: String?
    let timestamp: Date

    var photoCount: Int {
        photoIds.count
    }

    var formattedTimestamp: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        return formatter.string(from: timestamp)
    }

    var browserInfo: String {
        guard let userAgent = userAgent else { return "Unknown browser" }

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
