import SwiftUI

@main
struct PhotoSyncApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var delegate
    let persistenceController = PersistenceController.shared

    @State private var authRequestToShow: AuthRequest?
    @State private var deleteRequestToShow: DeleteRequest?

    init() {
        // Perform one-time migrations (API key to Keychain)
        AppSettings.performMigrations()

        // Initialize logger early and log app launch
        Task {
            await Logger.shared.info("PhotoSync app launched")

            // Initialize AutoSyncManager to start monitoring photo library
            _ = await AutoSyncManager.shared
            await Logger.shared.info("AutoSyncManager initialized")
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
                            await Logger.shared.info("Setting authRequestToShow")
                        }
                        authRequestToShow = request
                        Task {
                            await Logger.shared.info("authRequestToShow is now: \(authRequestToShow?.id ?? "nil")")
                        }
                    } else {
                        Task {
                            await Logger.shared.error("Failed to cast notification.object to AuthRequest")
                        }
                    }
                }
                .sheet(item: $authRequestToShow) { request in
                    AuthRequestView(
                        request: request,
                        onApprove: {
                            Task {
                                await Logger.shared.info("User tapped APPROVE button for request: \(request.id)")
                                await NotificationService.shared.approveAuthRequest()
                                await Logger.shared.info("Approve completed, dismissing sheet")
                                await MainActor.run {
                                    authRequestToShow = nil
                                }
                            }
                        },
                        onDeny: {
                            Task {
                                await Logger.shared.info("User tapped DENY button for request: \(request.id)")
                                await NotificationService.shared.denyAuthRequest()
                                await Logger.shared.info("Deny completed, dismissing sheet")
                                await MainActor.run {
                                    authRequestToShow = nil
                                }
                            }
                        }
                    )
                }
                .onReceive(NotificationCenter.default.publisher(for: .showDeleteRequest)) { notification in
                    Task {
                        await Logger.shared.info("PhotoSyncApp received showDeleteRequest notification")
                    }
                    if let request = notification.object as? DeleteRequest {
                        Task {
                            await Logger.shared.info("Successfully cast to DeleteRequest - id: \(request.id), photoCount: \(request.photoCount)")
                        }
                        deleteRequestToShow = request
                    } else {
                        Task {
                            await Logger.shared.error("Failed to cast notification.object to DeleteRequest")
                        }
                    }
                }
                .sheet(item: $deleteRequestToShow) { request in
                    DeleteRequestView(
                        request: request,
                        onApprove: {
                            Task {
                                await Logger.shared.info("User tapped APPROVE button for delete request: \(request.id)")
                                await NotificationService.shared.approveDeleteRequest()
                                await Logger.shared.info("Delete approve completed, dismissing sheet")
                                await MainActor.run {
                                    deleteRequestToShow = nil
                                }
                            }
                        },
                        onDeny: {
                            Task {
                                await Logger.shared.info("User tapped DENY button for delete request: \(request.id)")
                                await NotificationService.shared.denyDeleteRequest()
                                await Logger.shared.info("Delete deny completed, dismissing sheet")
                                await MainActor.run {
                                    deleteRequestToShow = nil
                                }
                            }
                        }
                    )
                }
        }
    }
}

// MARK: - Notification Names

extension Notification.Name {
    static let showAuthRequest = Notification.Name("showAuthRequest")
    static let showDeleteRequest = Notification.Name("showDeleteRequest")
}
