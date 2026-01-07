import SwiftUI

@main
struct PhotoSyncApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var delegate
    let persistenceController = PersistenceController.shared

    @State private var showAuthRequest = false
    @State private var currentAuthRequest: AuthRequest?

    init() {
        // Perform one-time migrations (API key to Keychain)
        AppSettings.performMigrations()

        // Initialize logger early and log app launch
        Task {
            await Logger.shared.info("PhotoSync app launched")
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(\.managedObjectContext, persistenceController.container.viewContext)
                .onAppear {
                    Task {
                        await Logger.shared.info("Main view appeared")

                        // Register device if configured but not registered
                        if AppSettings.isConfigured && !AppSettings.isDeviceRegistered {
                            await NotificationService.shared.registerDeviceWithServer()
                        }
                    }
                }
                .onReceive(NotificationCenter.default.publisher(for: .showAuthRequest)) { notification in
                    Task {
                        await Logger.shared.info("PhotoSyncApp received showAuthRequest notification")
                        await Logger.shared.info("Notification object type: \(type(of: notification.object))")
                    }
                    if let request = notification.object as? AuthRequest {
                        Task {
                            await Logger.shared.info("Successfully cast to AuthRequest - id: \(request.id), email: \(request.email)")
                        }
                        currentAuthRequest = request
                        showAuthRequest = true
                        Task {
                            await Logger.shared.info("Set currentAuthRequest and showAuthRequest=true")
                        }
                    } else {
                        Task {
                            await Logger.shared.error("Failed to cast notification.object to AuthRequest")
                        }
                    }
                }
                .sheet(isPresented: $showAuthRequest) {
                    if let request = currentAuthRequest {
                        AuthRequestView(
                            request: request,
                            onApprove: {
                                Task {
                                    await Logger.shared.info("User tapped APPROVE button for request: \(request.id)")
                                    await NotificationService.shared.approveAuthRequest()
                                    await Logger.shared.info("Approve completed, dismissing sheet")
                                    await MainActor.run {
                                        showAuthRequest = false
                                        currentAuthRequest = nil
                                    }
                                }
                            },
                            onDeny: {
                                Task {
                                    await Logger.shared.info("User tapped DENY button for request: \(request.id)")
                                    await NotificationService.shared.denyAuthRequest()
                                    await Logger.shared.info("Deny completed, dismissing sheet")
                                    await MainActor.run {
                                        showAuthRequest = false
                                        currentAuthRequest = nil
                                    }
                                }
                            }
                        )
                    }
                }
        }
    }
}

// MARK: - Notification Names

extension Notification.Name {
    static let showAuthRequest = Notification.Name("showAuthRequest")
}
