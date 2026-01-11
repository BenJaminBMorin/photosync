import SwiftUI

struct LoginView: View {
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var showPasswordReset = false

    var isValidForm: Bool {
        !email.isEmpty && !password.isEmpty && email.contains("@")
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                // Header
                VStack(spacing: 8) {
                    Image(systemName: "lock.fill")
                        .font(.system(size: 48))
                        .foregroundColor(.blue)

                    Text("Sign In")
                        .font(.title2)
                        .fontWeight(.bold)

                    Text("Connecting to:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    Text(AppSettings.normalizedServerURL)
                        .font(.caption)
                        .foregroundColor(.blue)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
                .padding(.top, 32)

                // Form
                VStack(spacing: 16) {
                    // Email
                    VStack(alignment: .leading, spacing: 4) {
                        Label("Email", systemImage: "envelope")
                            .font(.subheadline)
                            .fontWeight(.medium)

                        TextField("your@email.com", text: $email)
                            .textContentType(.emailAddress)
                            .keyboardType(.emailAddress)
                            .autocapitalization(.none)
                            .autocorrectionDisabled()
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                    }

                    // Password
                    VStack(alignment: .leading, spacing: 4) {
                        Label("Password", systemImage: "lock")
                            .font(.subheadline)
                            .fontWeight(.medium)

                        SecureField("Enter password", text: $password)
                            .textContentType(.password)
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                    }
                }
                .padding()

                // Error Message
                if showError {
                    HStack {
                        Image(systemName: "exclamationmark.circle.fill")
                            .foregroundColor(.red)
                        Text(errorMessage)
                            .font(.subheadline)
                        Spacer()
                    }
                    .padding()
                    .background(Color.red.opacity(0.1))
                    .cornerRadius(8)
                    .padding(.horizontal)
                }

                // Login Button
                Button(action: login) {
                    if isLoading {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                    } else {
                        Text("Login")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(isValidForm ? Color.blue : Color.gray)
                .foregroundColor(.white)
                .cornerRadius(8)
                .disabled(!isValidForm || isLoading)
                .padding(.horizontal)

                // Forgot Password
                Button("Forgot Password?") {
                    showPasswordReset = true
                }
                .font(.subheadline)
                .foregroundColor(.blue)

                Spacer()

                // Or use API Key
                Divider()
                    .padding(.horizontal)

                Text("Or configure manually in Settings")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
            .sheet(isPresented: $showPasswordReset) {
                PasswordResetView()
            }
        }
    }

    private func login() {
        isLoading = true
        errorMessage = ""
        showError = false

        Task {
            do {
                let fcmToken = await NotificationService.shared.getFCMToken()
                let deviceName = await MainActor.run { UIDevice.current.name }

                let response = try await APIService.shared.login(
                    email: email,
                    password: password,
                    deviceName: deviceName,
                    fcmToken: fcmToken ?? "unknown"
                )

                // Store API key, device info, and user email
                await MainActor.run {
                    AppSettings.apiKey = response.apiKey
                    AppSettings.deviceId = response.device.id
                    AppSettings.userEmail = response.user.email

                    // Record authentication time (to prevent race condition with stale 401s)
                    AppSettings.recordAuthentication()

                    // Clear form and dismiss
                    email = ""
                    password = ""
                    isLoading = false

                    dismiss()
                }

                await Logger.shared.info("Login successful for user: \(response.user.email)")
            } catch let error as APIError {
                await MainActor.run {
                    switch error {
                    case .unauthorized:
                        errorMessage = "Invalid email or password"
                    default:
                        errorMessage = error.localizedDescription ?? "Login failed"
                    }
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Login failed: \(error)")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Login failed: \(error)")
            }
        }
    }
}

#Preview {
    LoginView()
}
