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

        // Check if this is an auth request notification
        if let authRequestId = userInfo["requestId"] as? String {
            Task {
                await Logger.shared.info("Received auth request notification: \(authRequestId)")
                await NotificationService.shared.handleAuthRequest(
                    id: authRequestId,
                    userInfo: userInfo
                )
            }
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

        if let authRequestId = userInfo["requestId"] as? String {
            Task {
                await Logger.shared.info("User tapped auth request notification: \(authRequestId)")
                await NotificationService.shared.showAuthRequestUI(
                    id: authRequestId,
                    userInfo: userInfo
                )
            }
        }

        completionHandler()
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
