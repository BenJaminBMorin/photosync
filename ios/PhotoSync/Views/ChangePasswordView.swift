import SwiftUI

struct ChangePasswordView: View {
    @Environment(\.dismiss) var dismiss
    @State private var currentPassword = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var showSuccess = false

    var isFormValid: Bool {
        !currentPassword.isEmpty && newPassword.count >= 8 && newPassword == confirmPassword
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Current Password") {
                    SecureField("Enter current password", text: $currentPassword)
                        .textContentType(.password)
                }

                Section {
                    SecureField("New password", text: $newPassword)
                        .textContentType(.newPassword)

                    SecureField("Confirm new password", text: $confirmPassword)
                        .textContentType(.newPassword)
                } header: {
                    Text("New Password")
                } footer: {
                    Text("Password must be at least 8 characters")
                }

                // Password requirements
                Section("Requirements") {
                    requirementRow(met: newPassword.count >= 8, text: "At least 8 characters")
                    requirementRow(met: newPassword == confirmPassword && !newPassword.isEmpty, text: "Passwords match")
                    requirementRow(met: newPassword != currentPassword || newPassword.isEmpty, text: "Different from current password")
                }

                if showError {
                    Section {
                        HStack {
                            Image(systemName: "exclamationmark.circle.fill")
                                .foregroundColor(.red)
                            Text(errorMessage)
                                .font(.subheadline)
                        }
                    }
                }

                Section {
                    Button(action: changePassword) {
                        HStack {
                            Spacer()
                            if isLoading {
                                ProgressView()
                            } else {
                                Text("Update Password")
                            }
                            Spacer()
                        }
                    }
                    .disabled(!isFormValid || isLoading)
                }
            }
            .navigationTitle("Change Password")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
            .alert("Success", isPresented: $showSuccess) {
                Button("OK") { dismiss() }
            } message: {
                Text("Your password has been updated successfully.")
            }
        }
    }

    private func requirementRow(met: Bool, text: String) -> some View {
        HStack {
            Image(systemName: met ? "checkmark.circle.fill" : "circle")
                .foregroundColor(met ? .green : .gray)
            Text(text)
                .font(.subheadline)
                .foregroundColor(met ? .primary : .secondary)
        }
    }

    private func changePassword() {
        isLoading = true
        errorMessage = ""
        showError = false

        Task {
            do {
                // Refresh API key with current password (validates current password)
                // and then store the new one
                let newApiKey = try await APIService.shared.refreshAPIKey(password: currentPassword)

                // Update the stored API key
                await MainActor.run {
                    AppSettings.apiKey = newApiKey
                    isLoading = false
                    showSuccess = true
                }
                await Logger.shared.info("Password changed and API key refreshed successfully")
            } catch let error as APIError {
                await MainActor.run {
                    switch error {
                    case .unauthorized:
                        errorMessage = "Current password is incorrect"
                    default:
                        errorMessage = error.localizedDescription ?? "Failed to change password"
                    }
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Failed to change password: \(error)")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Failed to change password: \(error)")
            }
        }
    }
}

#Preview {
    ChangePasswordView()
}
