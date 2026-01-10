import SwiftUI

struct EmailPasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var currentStep: Step = .enterEmail
    @State private var email = ""
    @State private var code = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false

    enum Step: CaseIterable {
        case enterEmail
        case enterCode
        case enterNewPassword
        case success
    }

    var body: some View {
        VStack(spacing: 24) {
            // Progress indicator
            HStack(spacing: 8) {
                ForEach(Step.allCases, id: \.self) { step in
                    Circle()
                        .fill(isStepCompleted(step) ? Color.blue : Color.gray.opacity(0.3))
                        .frame(width: 8, height: 8)
                }
            }
            .padding()

            VStack(spacing: 16) {
                switch currentStep {
                case .enterEmail:
                    emailStep
                case .enterCode:
                    codeStep
                case .enterNewPassword:
                    passwordStep
                case .success:
                    successStep
                }
            }

            Spacer()
        }
        .padding()
        .navigationTitle("Email Reset")
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(currentStep != .enterEmail && currentStep != .success)
        .toolbar {
            if currentStep != .enterEmail && currentStep != .success {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Back") {
                        withAnimation {
                            goBack()
                        }
                    }
                }
            }
        }
    }

    private var emailStep: some View {
        VStack(spacing: 16) {
            Image(systemName: "envelope.fill")
                .font(.system(size: 48))
                .foregroundColor(.blue)

            Text("Enter your email address")
                .font(.headline)

            Text("We'll send a 6-digit code to reset your password")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)

            TextField("your@email.com", text: $email)
                .textContentType(.emailAddress)
                .keyboardType(.emailAddress)
                .autocapitalization(.none)
                .autocorrectionDisabled()
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)

            if showError {
                errorView
            }

            Button(action: sendResetCode) {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Text("Send Code")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(email.contains("@") ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!email.contains("@") || isLoading)
        }
    }

    private var codeStep: some View {
        VStack(spacing: 16) {
            Image(systemName: "number.circle.fill")
                .font(.system(size: 48))
                .foregroundColor(.blue)

            Text("Enter verification code")
                .font(.headline)

            Text("We sent a code to \(email)")
                .font(.subheadline)
                .foregroundColor(.secondary)

            TextField("000000", text: $code)
                .keyboardType(.numberPad)
                .multilineTextAlignment(.center)
                .font(.system(size: 32, weight: .bold, design: .monospaced))
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
                .onChange(of: code) { _, newValue in
                    // Limit to 6 digits
                    if newValue.count > 6 {
                        code = String(newValue.prefix(6))
                    }
                    // Remove non-digits
                    code = newValue.filter { $0.isNumber }
                }

            Text("Code expires in 15 minutes")
                .font(.caption)
                .foregroundColor(.orange)

            if showError {
                errorView
            }

            Button(action: proceedToPassword) {
                Text("Continue")
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(code.count == 6 ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(code.count != 6)

            Button("Resend Code") {
                sendResetCode()
            }
            .font(.subheadline)
            .foregroundColor(.blue)
        }
    }

    private var passwordStep: some View {
        VStack(spacing: 16) {
            Image(systemName: "lock.fill")
                .font(.system(size: 48))
                .foregroundColor(.blue)

            Text("Create new password")
                .font(.headline)

            Text("Password must be at least 8 characters")
                .font(.subheadline)
                .foregroundColor(.secondary)

            SecureField("New password", text: $newPassword)
                .textContentType(.newPassword)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)

            SecureField("Confirm password", text: $confirmPassword)
                .textContentType(.newPassword)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)

            // Password requirements
            VStack(alignment: .leading, spacing: 4) {
                requirementRow(met: newPassword.count >= 8, text: "At least 8 characters")
                requirementRow(met: newPassword == confirmPassword && !newPassword.isEmpty, text: "Passwords match")
            }
            .padding()
            .background(Color(.systemGray6))
            .cornerRadius(8)

            if showError {
                errorView
            }

            Button(action: resetPassword) {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Text("Reset Password")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isPasswordValid ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!isPasswordValid || isLoading)
        }
    }

    private var successStep: some View {
        VStack(spacing: 24) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)

            VStack(spacing: 8) {
                Text("Password Reset!")
                    .font(.title2)
                    .fontWeight(.bold)

                Text("Your password has been successfully reset. You can now login with your new password.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }

            Button(action: { dismiss() }) {
                Text("Done")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
    }

    private var errorView: some View {
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
    }

    private func requirementRow(met: Bool, text: String) -> some View {
        HStack {
            Image(systemName: met ? "checkmark.circle.fill" : "circle")
                .foregroundColor(met ? .green : .gray)
            Text(text)
                .font(.caption)
                .foregroundColor(met ? .primary : .secondary)
        }
    }

    private var isPasswordValid: Bool {
        newPassword.count >= 8 && newPassword == confirmPassword
    }

    private func isStepCompleted(_ step: Step) -> Bool {
        switch step {
        case .enterEmail:
            return currentStep != .enterEmail
        case .enterCode:
            return currentStep == .enterNewPassword || currentStep == .success
        case .enterNewPassword:
            return currentStep == .success
        case .success:
            return currentStep == .success
        }
    }

    private func goBack() {
        showError = false
        errorMessage = ""
        switch currentStep {
        case .enterCode:
            currentStep = .enterEmail
        case .enterNewPassword:
            currentStep = .enterCode
        default:
            break
        }
    }

    private func sendResetCode() {
        isLoading = true
        errorMessage = ""
        showError = false

        Task {
            do {
                try await APIService.shared.initiateEmailPasswordReset(email: email)
                await MainActor.run {
                    withAnimation {
                        currentStep = .enterCode
                    }
                    isLoading = false
                }
                await Logger.shared.info("Password reset code sent to: \(email)")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Failed to send reset code: \(error)")
            }
        }
    }

    private func proceedToPassword() {
        showError = false
        withAnimation {
            currentStep = .enterNewPassword
        }
    }

    private func resetPassword() {
        isLoading = true
        errorMessage = ""
        showError = false

        Task {
            do {
                try await APIService.shared.verifyPasswordResetCode(
                    email: email,
                    code: code,
                    newPassword: newPassword
                )
                await MainActor.run {
                    withAnimation {
                        currentStep = .success
                    }
                    isLoading = false
                }
                await Logger.shared.info("Password reset successful for: \(email)")
            } catch let error as APIError {
                await MainActor.run {
                    switch error {
                    case .unauthorized:
                        errorMessage = "Invalid or expired code. Please try again."
                        currentStep = .enterCode
                        code = ""
                    default:
                        errorMessage = error.localizedDescription ?? "Failed to reset password"
                    }
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Password reset failed: \(error)")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Password reset failed: \(error)")
            }
        }
    }
}

#Preview {
    NavigationStack {
        EmailPasswordResetView()
    }
}
