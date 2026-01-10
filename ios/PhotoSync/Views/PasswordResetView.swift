import SwiftUI

struct PasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var selectedMethod: ResetMethod = .email

    enum ResetMethod {
        case email
        case phone
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                // Header
                VStack(spacing: 8) {
                    Image(systemName: "key.fill")
                        .font(.system(size: 48))
                        .foregroundColor(.orange)

                    Text("Reset Password")
                        .font(.title2)
                        .fontWeight(.bold)

                    Text("Choose a reset method")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 32)

                // Method Selector
                Picker("Reset Method", selection: $selectedMethod) {
                    Text("Email with Code").tag(ResetMethod.email)
                    Text("Phone 2FA").tag(ResetMethod.phone)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)

                // Method-specific content
                VStack(spacing: 16) {
                    if selectedMethod == .email {
                        NavigationLink(destination: EmailPasswordResetView()) {
                            HStack {
                                Image(systemName: "envelope.fill")
                                    .foregroundColor(.blue)
                                    .frame(width: 32)
                                VStack(alignment: .leading) {
                                    Text("Email Reset")
                                        .fontWeight(.semibold)
                                        .foregroundColor(.primary)
                                    Text("Get a code sent to your email")
                                        .font(.subheadline)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .foregroundColor(.gray)
                            }
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(12)
                        }
                    } else {
                        NavigationLink(destination: PhonePasswordResetView()) {
                            HStack {
                                Image(systemName: "iphone")
                                    .foregroundColor(.blue)
                                    .frame(width: 32)
                                VStack(alignment: .leading) {
                                    Text("Phone 2FA")
                                        .fontWeight(.semibold)
                                        .foregroundColor(.primary)
                                    Text("Approve on another device")
                                        .font(.subheadline)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .foregroundColor(.gray)
                            }
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(12)
                        }
                    }
                }
                .padding(.horizontal)

                // Instructions
                VStack(alignment: .leading, spacing: 8) {
                    if selectedMethod == .email {
                        HStack(alignment: .top) {
                            Image(systemName: "info.circle")
                                .foregroundColor(.blue)
                            Text("A 6-digit code will be sent to your email. Enter the code to reset your password.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    } else {
                        HStack(alignment: .top) {
                            Image(systemName: "info.circle")
                                .foregroundColor(.blue)
                            Text("A push notification will be sent to your registered devices. Approve the request to reset your password.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
                .padding(.horizontal)

                Spacer()
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}

#Preview {
    PasswordResetView()
}
