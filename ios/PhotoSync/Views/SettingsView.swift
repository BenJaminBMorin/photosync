import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @State private var showAPIKey = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Server Configuration") {
                    TextField("Server URL", text: $viewModel.serverURL)
                        .textContentType(.URL)
                        .keyboardType(.URL)
                        .autocapitalization(.none)
                        .autocorrectionDisabled()

                    HStack {
                        if showAPIKey {
                            TextField("API Key", text: $viewModel.apiKey)
                                .autocapitalization(.none)
                                .autocorrectionDisabled()
                        } else {
                            SecureField("API Key", text: $viewModel.apiKey)
                        }

                        Button {
                            showAPIKey.toggle()
                        } label: {
                            Image(systemName: showAPIKey ? "eye.slash" : "eye")
                                .foregroundColor(.secondary)
                        }
                        .buttonStyle(.plain)
                    }

                    Button {
                        Task {
                            await viewModel.testConnection()
                        }
                    } label: {
                        HStack {
                            if viewModel.isTesting {
                                ProgressView()
                                    .scaleEffect(0.8)
                                Text("Testing...")
                            } else {
                                Image(systemName: "network")
                                Text("Test Connection")
                            }
                        }
                    }
                    .disabled(!viewModel.isConfigured || viewModel.isTesting)

                    if let result = viewModel.testResult {
                        testResultView(result)
                    }
                }

                Section("Sync Settings") {
                    Toggle(isOn: $viewModel.wifiOnly) {
                        VStack(alignment: .leading) {
                            Text("Wi-Fi Only")
                            Text("Only sync when connected to Wi-Fi")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    Toggle(isOn: $viewModel.autoSync) {
                        VStack(alignment: .leading) {
                            Text("Auto-Sync New Photos")
                            Text("Automatically sync new photos when app is available")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    Button {
                        Task {
                            await viewModel.resyncFromServer()
                        }
                    } label: {
                        HStack {
                            if viewModel.isResyncing {
                                ProgressView()
                                    .scaleEffect(0.8)
                                Text("Resyncing...")
                            } else {
                                Image(systemName: "arrow.triangle.2.circlepath")
                                Text("Resync from Server")
                            }
                        }
                    }
                    .disabled(!viewModel.isConfigured || viewModel.isResyncing)

                    if let result = viewModel.resyncResult {
                        resyncResultView(result)
                    }
                }

                Section("Display Settings") {
                    Toggle(isOn: $viewModel.showServerOnlyPhotos) {
                        VStack(alignment: .leading) {
                            Text("Show Server-Only Photos")
                            Text("Display photos on server but not on phone")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                }

                Section("Debugging") {
                    NavigationLink {
                        LogsView()
                    } label: {
                        HStack {
                            Image(systemName: "doc.text")
                            Text("View Logs")
                        }
                    }
                }

                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("1.0.0")
                            .foregroundColor(.secondary)
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text("PhotoSync")
                            .font(.headline)
                        Text("Sync your photos to your NAS server. Photos are organized by Year/Month folders on the server.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding(.vertical, 4)
                }
            }
            .navigationTitle("Settings")
        }
    }

    @ViewBuilder
    private func testResultView(_ result: SettingsViewModel.TestResult) -> some View {
        switch result {
        case .success:
            HStack {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundColor(.green)
                Text("Connection successful!")
                    .foregroundColor(.green)
            }

        case .failure(let message):
            HStack {
                Image(systemName: "xmark.circle.fill")
                    .foregroundColor(.red)
                Text(message)
                    .foregroundColor(.red)
                    .font(.caption)
            }
        }
    }

    @ViewBuilder
    private func resyncResultView(_ result: SettingsViewModel.ResyncResult) -> some View {
        switch result {
        case .success(let count):
            HStack {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundColor(.green)
                Text("Resync complete!")
                    .foregroundColor(.green)
            }

        case .failure(let message):
            HStack {
                Image(systemName: "xmark.circle.fill")
                    .foregroundColor(.red)
                Text(message)
                    .foregroundColor(.red)
                    .font(.caption)
            }
        }
    }
}

#Preview {
    SettingsView()
}
