import SwiftUI

struct PasswordResetApprovalView: View {
    let request: PasswordResetRequest
    let onApprove: () -> Void
    let onDeny: () -> Void

    @State private var isApproving = false
    @State private var isDenying = false

    var body: some View {
        VStack(spacing: 24) {
            // Warning icon
            Image(systemName: "exclamationmark.shield.fill")
                .font(.system(size: 64))
                .foregroundColor(.orange)

            // Title
            Text("Password Reset Request")
                .font(.title2)
                .fontWeight(.bold)

            // Request details
            VStack(spacing: 12) {
                detailRow(icon: "envelope.fill", title: "Account", value: request.email)
                detailRow(icon: "globe", title: "IP Address", value: request.ipAddress ?? "Unknown")
                detailRow(icon: "desktopcomputer", title: "Device", value: request.browserInfo)
                detailRow(icon: "clock.fill", title: "Time", value: request.formattedTimestamp)
            }
            .padding()
            .background(Color(.systemGray6))
            .cornerRadius(12)

            // Warning message
            VStack(spacing: 8) {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.orange)
                    Text("Security Notice")
                        .fontWeight(.semibold)
                }

                Text("Someone is trying to reset the password for this account. Only approve if you initiated this request.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding()
            .background(Color.orange.opacity(0.1))
            .cornerRadius(12)

            Spacer()

            // Action buttons
            VStack(spacing: 12) {
                Button {
                    isApproving = true
                    onApprove()
                } label: {
                    HStack {
                        if isApproving {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Image(systemName: "checkmark.circle.fill")
                            Text("Approve Reset")
                        }
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(.green)
                .controlSize(.large)
                .disabled(isApproving || isDenying)

                Button {
                    isDenying = true
                    onDeny()
                } label: {
                    HStack {
                        if isDenying {
                            ProgressView()
                        } else {
                            Image(systemName: "xmark.circle.fill")
                            Text("Deny Request")
                        }
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(.red)
                .controlSize(.large)
                .disabled(isApproving || isDenying)
            }
        }
        .padding()
    }

    private func detailRow(icon: String, title: String, value: String) -> some View {
        HStack {
            Image(systemName: icon)
                .foregroundColor(.blue)
                .frame(width: 24)
            Text(title)
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .fontWeight(.medium)
        }
    }
}

#Preview {
    PasswordResetApprovalView(
        request: PasswordResetRequest(
            id: "test-123",
            email: "user@example.com",
            ipAddress: "192.168.1.100",
            userAgent: "iPhone",
            timestamp: Date()
        ),
        onApprove: {},
        onDeny: {}
    )
}
