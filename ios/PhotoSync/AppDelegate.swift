import UIKit
import UserNotifications
import FirebaseCore
import FirebaseMessaging

class AppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        // Configure Firebase
        FirebaseApp.configure()

        // Set up push notifications
        UNUserNotificationCenter.current().delegate = self

        // Set up FCM delegate
        Messaging.messaging().delegate = self

        // Request notification permissions
        requestNotificationPermissions()

        // Register for remote notifications
        application.registerForRemoteNotifications()

        return true
    }

    private func requestNotificationPermissions() {
        let authOptions: UNAuthorizationOptions = [.alert, .badge, .sound]
        UNUserNotificationCenter.current().requestAuthorization(options: authOptions) { granted, error in
            if let error = error {
                Task {
                    await Logger.shared.error("Failed to request notification permissions: \(error)")
                }
            } else {
                Task {
                    await Logger.shared.info("Notification permissions granted: \(granted)")
                }
            }
        }
    }

    // MARK: - Remote Notification Registration

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        // Pass the APNs token to Firebase
        Messaging.messaging().apnsToken = deviceToken

        let tokenString = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        Task {
            await Logger.shared.info("APNs token received: \(tokenString.prefix(20))...")
        }
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        Task {
            await Logger.shared.error("Failed to register for remote notifications: \(error)")
        }
    }

    // MARK: - UNUserNotificationCenterDelegate

    // Handle notification when app is in foreground
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        let userInfo = notification.request.content.userInfo

        Task {
            await handleNotification(userInfo: userInfo)
        }

        // Show the notification even when app is in foreground
        completionHandler([.banner, .sound, .badge])
    }

    // Handle notification tap
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let userInfo = response.notification.request.content.userInfo

        Task {
            await handleNotification(userInfo: userInfo, tapped: true)
        }

        completionHandler()
    }

    // MARK: - Notification Handling

    private func handleNotification(userInfo: [AnyHashable: Any], tapped: Bool = false) async {
        guard let type = userInfo["type"] as? String else {
            await Logger.shared.warning("Received notification without type field")
            return
        }

        await Logger.shared.info("Handling notification of type: \(type), tapped: \(tapped)")

        switch type {
        case "auth_request":
            await handleAuthRequestNotification(userInfo: userInfo, tapped: tapped)
        case "delete_request":
            await handleDeleteRequestNotification(userInfo: userInfo, tapped: tapped)
        default:
            await Logger.shared.warning("Unknown notification type: \(type)")
        }
    }

    private func handleAuthRequestNotification(userInfo: [AnyHashable: Any], tapped: Bool) async {
        guard let requestId = userInfo["requestId"] as? String else {
            await Logger.shared.error("Auth request notification missing requestId")
            return
        }

        await Logger.shared.info("Handling auth request: \(requestId), tapped: \(tapped)")

        if tapped {
            // User tapped the notification - show UI immediately
            await NotificationService.shared.showAuthRequestUI(
                id: requestId,
                userInfo: userInfo
            )
        } else {
            // Received while app is open - just handle it (will show UI if needed)
            await NotificationService.shared.handleAuthRequest(
                id: requestId,
                userInfo: userInfo
            )
        }
    }

    private func handleDeleteRequestNotification(userInfo: [AnyHashable: Any], tapped: Bool) async {
        guard let requestId = userInfo["requestId"] as? String else {
            await Logger.shared.error("Delete request notification missing requestId")
            return
        }

        await Logger.shared.info("Handling delete request: \(requestId), tapped: \(tapped)")

        if tapped {
            await NotificationService.shared.showDeleteRequestUI(
                id: requestId,
                userInfo: userInfo
            )
        } else {
            await NotificationService.shared.handleDeleteRequest(
                id: requestId,
                userInfo: userInfo
            )
        }
    }
}

// MARK: - MessagingDelegate

extension AppDelegate: MessagingDelegate {
    func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
        guard let token = fcmToken else { return }

        Task {
            await Logger.shared.info("FCM token received: \(token.prefix(20))...")
            await NotificationService.shared.updateFCMToken(token)
        }
    }
}
