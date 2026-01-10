import SwiftUI

@main
struct PhotoSyncApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var delegate
    let persistenceController = PersistenceController.shared

    @State private var authRequestToShow: AuthRequest?
    @State private var deleteRequestToShow: DeleteRequest?
    @State private var passwordResetRequestToShow: PasswordResetRequest?
    @State private var showInviteError: Bool = false
    @State private var inviteErrorMessage: String = ""
    @State private var isProcessingInvite: Bool = false
    @State private var showInviteSuccess: Bool = false
    @State private var inviteSuccessEmail: String = ""
    @State private var showAuthenticationRequired: Bool = false

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
                .onReceive(NotificationCenter.default.publisher(for: .showPasswordResetRequest)) { notification in
                    Task {
                        await Logger.shared.info("PhotoSyncApp received showPasswordResetRequest notification")
                    }
                    if let request = notification.object as? PasswordResetRequest {
                        Task {
                            await Logger.shared.info("Successfully cast to PasswordResetRequest - id: \(request.id), email: \(request.email)")
                        }
                        passwordResetRequestToShow = request
                    } else {
                        Task {
                            await Logger.shared.error("Failed to cast notification.object to PasswordResetRequest")
                        }
                    }
                }
                .sheet(item: $passwordResetRequestToShow) { request in
                    PasswordResetApprovalView(
                        request: request,
                        onApprove: {
                            Task {
                                await Logger.shared.info("User tapped APPROVE button for password reset: \(request.id)")
                                await NotificationService.shared.approvePasswordResetRequest()
                                await Logger.shared.info("Password reset approve completed, dismissing sheet")
                                await MainActor.run {
                                    passwordResetRequestToShow = nil
                                }
                            }
                        },
                        onDeny: {
                            Task {
                                await Logger.shared.info("User tapped DENY button for password reset: \(request.id)")
                                await NotificationService.shared.denyPasswordResetRequest()
                                await Logger.shared.info("Password reset deny completed, dismissing sheet")
                                await MainActor.run {
                                    passwordResetRequestToShow = nil
                                }
                            }
                        }
                    )
                }
                .onReceive(NotificationCenter.default.publisher(for: .authenticationRequired)) { _ in
                    Task {
                        await Logger.shared.warning("Authentication required - prompting user")
                    }
                    showAuthenticationRequired = true
                }
                .onOpenURL { url in
                    Task {
                        await handleDeepLink(url)
                    }
                }
                .alert("Session Expired", isPresented: $showAuthenticationRequired) {
                    Button("Go to Settings") {
                        showAuthenticationRequired = false
                    }
                } message: {
                    Text("Your session has expired. Please sign in again in Settings.")
                }
                .alert("Invite Error", isPresented: $showInviteError) {
                    Button("OK", role: .cancel) {
                        showInviteError = false
                        inviteErrorMessage = ""
                    }
                } message: {
                    Text(inviteErrorMessage)
                }
                .alert("Welcome to PhotoSync!", isPresented: $showInviteSuccess) {
                    Button("Get Started", role: .none) {
                        showInviteSuccess = false
                        inviteSuccessEmail = ""
                    }
                } message: {
                    Text("Your account (\(inviteSuccessEmail)) has been configured. You can now start syncing photos to your server.")
                }
                .overlay {
                    if isProcessingInvite {
                        ZStack {
                            Color.black.opacity(0.4)
                                .ignoresSafeArea()

                            VStack(spacing: 16) {
                                ProgressView()
                                    .scaleEffect(1.5)
                                    .tint(.white)

                                Text("Setting up your account...")
                                    .foregroundColor(.white)
                                    .font(.headline)
                            }
                            .padding(32)
                            .background(Color.black.opacity(0.8))
                            .cornerRadius(16)
                        }
                    }
                }
        }
    }

    /// Handle deep link URLs (photosync://invite?token=...)
    private func handleDeepLink(_ url: URL) async {
        await Logger.shared.info("Received deep link: \(url.absoluteString)")

        // Check if this is an invite link
        guard url.scheme == "photosync",
              url.host == "invite" else {
            await Logger.shared.warning("Unknown deep link scheme/host: \(url.absoluteString)")
            return
        }

        // Extract token parameter
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let queryItems = components.queryItems,
              let tokenItem = queryItems.first(where: { $0.name == "token" }),
              let token = tokenItem.value else {
            await Logger.shared.error("Invite link missing token parameter")
            await MainActor.run {
                inviteErrorMessage = "Invalid invite link - missing token"
                showInviteError = true
            }
            return
        }

        await Logger.shared.info("Extracted invite token (length: \(token.count))")

        // Set processing flag
        await MainActor.run {
            isProcessingInvite = true
        }

        do {
            // Redeem the invite token
            let response = try await InviteService.shared.redeemInvite(token: token)

            await Logger.shared.info("Invite redeemed successfully for: \(response.email)")

            // Success - credentials are already saved by InviteService
            // Show success message
            await MainActor.run {
                isProcessingInvite = false
                inviteSuccessEmail = response.email
                showInviteSuccess = true
            }

        } catch let error as InviteError {
            await Logger.shared.error("Invite redemption failed: \(error.localizedDescription)")
            await MainActor.run {
                isProcessingInvite = false
                inviteErrorMessage = error.localizedDescription ?? "Failed to redeem invite"
                showInviteError = true
            }
        } catch {
            await Logger.shared.error("Unexpected error redeeming invite: \(error)")
            await MainActor.run {
                isProcessingInvite = false
                inviteErrorMessage = "An unexpected error occurred"
                showInviteError = true
            }
        }
    }
}

// MARK: - Notification Names

extension Notification.Name {
    static let showAuthRequest = Notification.Name("showAuthRequest")
    static let showDeleteRequest = Notification.Name("showDeleteRequest")
    static let showPasswordResetRequest = Notification.Name("showPasswordResetRequest")
}
