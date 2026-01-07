import SwiftUI

/// View for approving or denying photo deletion requests
struct DeleteRequestView: View {
    let request: DeleteRequest
    let onApprove: () -> Void
    let onDeny: () -> Void

    @State private var isProcessing = false

    var body: some View {
        VStack(spacing: 24) {
            // Header
            VStack(spacing: 8) {
                Image(systemName: "trash.fill")
                    .font(.system(size: 48))
                    .foregroundColor(.red)
                    .onAppear {
                        Task {
                            await Logger.shared.info("DeleteRequestView appeared - request.id: \(request.id), email: \(request.email)")
                        }
                    }

                Text("Delete Request")
                    .font(.title2)
                    .fontWeight(.bold)
            }
            .padding(.top, 24)

            // Request details
            VStack(alignment: .leading, spacing: 16) {
                DetailRow(icon: "envelope.fill", label: "Account", value: request.email)

                DetailRow(icon: "photo.fill", label: "Photos", value: "\(request.photoCount)")

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
            VStack(spacing: 8) {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.red)
                    Text("Only approve if you initiated this deletion")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                Text("This action cannot be undone")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .fontWeight(.semibold)
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
                        Text("Approve Deletion")
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.red)
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

// MARK: - Preview

#Preview {
    DeleteRequestView(
        request: DeleteRequest(
            id: "test-123",
            photoIds: ["photo1", "photo2", "photo3"],
            email: "user@example.com",
            ipAddress: "192.168.1.1",
            userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
            timestamp: Date()
        ),
        onApprove: { print("Approved") },
        onDeny: { print("Denied") }
    )
}
