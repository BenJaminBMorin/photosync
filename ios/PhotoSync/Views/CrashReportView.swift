import SwiftUI

struct CrashReportView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var crashLog: String = ""
    let logURL: URL?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Header
                VStack(spacing: 12) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 60))
                        .foregroundColor(.red)

                    Text("App Crashed Previously")
                        .font(.title2)
                        .fontWeight(.bold)

                    Text("The app crashed during your last session. Here are the logs from that crash:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }
                .padding(.vertical, 24)
                .background(Color(.systemGroupedBackground))

                Divider()

                // Log content
                ScrollView {
                    if crashLog.isEmpty {
                        ContentUnavailableView(
                            "No Crash Log Available",
                            systemImage: "doc.text.magnifyingglass",
                            description: Text("Unable to load crash log from previous session")
                        )
                    } else {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Crash Log")
                                .font(.headline)
                                .padding(.horizontal)
                                .padding(.top)

                            Text(crashLog)
                                .font(.system(.caption, design: .monospaced))
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding()
                                .background(Color(.systemGray6))
                                .cornerRadius(8)
                                .padding(.horizontal)
                                .textSelection(.enabled)
                        }
                    }
                }

                Divider()

                // Actions
                VStack(spacing: 12) {
                    if let logURL = logURL {
                        ShareLink(item: logURL) {
                            Label("Share Crash Log", systemImage: "square.and.arrow.up")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .padding(.horizontal)
                    }

                    Button {
                        dismiss()
                    } label: {
                        Text("Continue")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .padding(.horizontal)
                }
                .padding(.vertical)
                .background(Color(.systemGroupedBackground))
            }
            .navigationBarTitleDisplayMode(.inline)
        }
        .task {
            if let logURL = logURL {
                crashLog = Logger.shared.getLogContent(from: logURL) ?? "Failed to load crash log"
            }
        }
    }
}

#Preview {
    CrashReportView(logURL: nil)
}
