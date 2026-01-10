import SwiftUI

struct PhonePasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var requestId = ""
    @State private var currentStep: Step = .enterCredentials
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var statusCheckTimer: Timer?
    @State private var approvalStatus = "Pending..."
    @State private var timeRemaining = 60

    enum Step {
        case enterCredentials
        case waiting
        case approved
    }

    var isFormValid: Bool {
        email.contains("@") && newPassword.count >= 8 && newPassword == confirmPassword
    }

    var body: some View {
        VStack(spacing: 24) {
            switch currentStep {
            case .enterCredentials:
                credentialsStep
            case .waiting:
                waitingStep
            case .approved:
                successStep
            }

            Spacer()
        }
        .padding()
        .navigationTitle("Phone 2FA Reset")
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(currentStep == .waiting)
        .toolbar {
            if currentStep == .waiting {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") {
                        cancelRequest()
                    }
                }
            }
        }
        .onDisappear {
            statusCheckTimer?.invalidate()
        }
    }

    private var credentialsStep: some View {
        VStack(spacing: 16) {
            Image(systemName: "iphone.radiowaves.left.and.right")
                .font(.system(size: 48))
                .foregroundColor(.blue)

            Text("Reset via Phone")
                .font(.headline)

            Text("Enter your credentials. An approval request will be sent to your registered devices.")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)

            VStack(spacing: 12) {
                TextField("Email", text: $email)
                    .textContentType(.emailAddress)
                    .keyboardType(.emailAddress)
                    .autocapitalization(.none)
                    .autocorrectionDisabled()
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(8)

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
            }

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

            Button(action: requestPhoneReset) {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Text("Send Approval Request")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isFormValid ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!isFormValid || isLoading)
        }
    }

    private var waitingStep: some View {
        VStack(spacing: 24) {
            // Animated waiting indicator
            VStack(spacing: 16) {
                Image(systemName: "iphone.radiowaves.left.and.right")
                    .font(.system(size: 48))
                    .foregroundColor(.blue)
                    .symbolEffect(.pulse, options: .repeating)

                Text("Waiting for Approval")
                    .font(.headline)

                Text("Check your other device for an approval request")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding()
            .background(Color.blue.opacity(0.1))
            .cornerRadius(12)

            // Countdown timer
            VStack(spacing: 8) {
                Text("Time remaining")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Text("\(timeRemaining)s")
                    .font(.system(size: 32, weight: .bold, design: .monospaced))
                    .foregroundColor(timeRemaining <= 10 ? .red : .primary)
            }

            // Status message
            HStack {
                Image(systemName: statusIcon)
                    .foregroundColor(statusColor)
                Text("Status: \(approvalStatus)")
                    .font(.subheadline)
            }
            .padding()
            .background(Color(.systemGray6))
            .cornerRadius(8)

            if showError {
                errorView
            }
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

    private var statusIcon: String {
        switch approvalStatus.lowercased() {
        case let s where s.contains("pending"):
            return "clock.fill"
        case let s where s.contains("approved"):
            return "checkmark.circle.fill"
        case let s where s.contains("denied"):
            return "xmark.circle.fill"
        default:
            return "info.circle.fill"
        }
    }

    private var statusColor: Color {
        switch approvalStatus.lowercased() {
        case let s where s.contains("pending"):
            return .orange
        case let s where s.contains("approved"):
            return .green
        case let s where s.contains("denied"):
            return .red
        default:
            return .blue
        }
    }

    private func requestPhoneReset() {
        isLoading = true
        errorMessage = ""
        showError = false

        Task {
            do {
                let id = try await APIService.shared.initiatePhonePasswordReset(
                    email: email,
                    newPassword: newPassword
                )

                await MainActor.run {
                    requestId = id
                    currentStep = .waiting
                    isLoading = false
                    approvalStatus = "Pending..."
                    timeRemaining = 60

                    startPollingForApproval()
                    startCountdownTimer()
                }
                await Logger.shared.info("Phone reset request initiated: \(id)")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
                await Logger.shared.error("Failed to initiate phone reset: \(error)")
            }
        }
    }

    private func startCountdownTimer() {
        Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { timer in
            if timeRemaining > 0 {
                timeRemaining -= 1
            } else {
                timer.invalidate()
            }
        }
    }

    private func startPollingForApproval() {
        statusCheckTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { _ in
            Task {
                do {
                    let status = try await APIService.shared.checkPhoneResetStatus(requestId: requestId)

                    await MainActor.run {
                        approvalStatus = status.status.capitalized

                        switch status.status.lowercased() {
                        case "approved":
                            statusCheckTimer?.invalidate()
                            completePhoneReset()
                        case "denied":
                            statusCheckTimer?.invalidate()
                            errorMessage = "Reset request was denied"
                            showError = true
                            currentStep = .enterCredentials
                        case "expired":
                            statusCheckTimer?.invalidate()
                            errorMessage = "Reset request has expired"
                            showError = true
                            currentStep = .enterCredentials
                        default:
                            break
                        }
                    }
                } catch {
                    // Silently handle polling errors
                    await Logger.shared.error("Polling error: \(error)")
                }
            }
        }
    }

    private func completePhoneReset() {
        Task {
            do {
                try await APIService.shared.completePhoneReset(requestId: requestId)

                await MainActor.run {
                    currentStep = .approved
                }
                await Logger.shared.info("Phone reset completed successfully")
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    currentStep = .enterCredentials
                }
                await Logger.shared.error("Failed to complete phone reset: \(error)")
            }
        }
    }

    private func cancelRequest() {
        statusCheckTimer?.invalidate()
        currentStep = .enterCredentials
        showError = false
        errorMessage = ""
    }
}

#Preview {
    NavigationStack {
        PhonePasswordResetView()
    }
}
