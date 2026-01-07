import SwiftUI

/// View for approving or denying web authentication requests
struct AuthRequestView: View {
    let request: AuthRequest
    let onApprove: () -> Void
    let onDeny: () -> Void

    @State private var isProcessing = false

    var body: some View {
        VStack(spacing: 24) {
            // Header
            VStack(spacing: 8) {
                Image(systemName: "person.badge.key.fill")
                    .font(.system(size: 48))
                    .foregroundColor(.blue)
                    .onAppear {
                        Task {
                            await Logger.shared.info("AuthRequestView appeared - request.id: \(request.id), email: \(request.email)")
                        }
                    }

                Text("Login Request")
                    .font(.title2)
                    .fontWeight(.bold)
            }
            .padding(.top, 24)

            // Request details
            VStack(alignment: .leading, spacing: 16) {
                DetailRow(icon: "envelope.fill", label: "Account", value: request.email)

                DetailRow(icon: "globe", label: "Browser", value: request.browserInfo)

                if let ip = request.ipAddress {
                    DetailRow(icon: "network", label: "IP Address", value: ip)
                }

                DetailRow(icon: "clock.fill", label: "Time", value: request.formattedTimestamp)
            }
            .padding()
            .background(Color(.systemGray6))
            .cornerRadius(12)

            // Warning
            HStack {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundColor(.orange)
                Text("Only approve if you initiated this login")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()

            // Action buttons
            VStack(spacing: 12) {
                Button(action: {
                    isProcessing = true
                    onApprove()
                }) {
                    HStack {
                        if isProcessing {
                            ProgressView()
                                .progressViewStyle(CircularProgressViewStyle(tint: .white))
                        } else {
                            Image(systemName: "checkmark.circle.fill")
                        }
                        Text("Approve Login")
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.green)
                    .foregroundColor(.white)
                    .cornerRadius(12)
                }
                .disabled(isProcessing)

                Button(action: {
                    isProcessing = true
                    onDeny()
                }) {
                    HStack {
                        Image(systemName: "xmark.circle.fill")
                        Text("Deny")
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color(.systemGray5))
                    .foregroundColor(.primary)
                    .cornerRadius(12)
                }
                .disabled(isProcessing)
            }
            .padding(.bottom, 24)
        }
        .padding()
        .background(Color(.systemBackground))
    }
}

struct DetailRow: View {
    let icon: String
    let label: String
    let value: String

    var body: some View {
        HStack {
            Image(systemName: icon)
                .foregroundColor(.secondary)
                .frame(width: 24)

            Text(label)
                .foregroundColor(.secondary)

            Spacer()

            Text(value)
                .fontWeight(.medium)
        }
    }
}

// MARK: - Preview

#Preview {
    AuthRequestView(
        request: AuthRequest(
            id: "test-123",
            email: "user@example.com",
            ipAddress: "192.168.1.1",
            userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
            timestamp: Date()
        ),
        onApprove: { print("Approved") },
        onDeny: { print("Denied") }
    )
}
