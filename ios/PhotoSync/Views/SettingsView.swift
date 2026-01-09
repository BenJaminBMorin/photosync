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
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("Resyncing...")
                                    if let progress = viewModel.resyncProgress {
                                        Text(progress)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                }
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

                    // Show sync status if available
                    if let syncStatus = viewModel.syncStatus {
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Image(systemName: "info.circle")
                                    .foregroundColor(.blue)
                                Text("Sync Status")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }

                            Text("\(syncStatus.totalPhotos) photos on server")
                                .font(.caption2)
                            Text("\(syncStatus.devicePhotos) from this device")
                                .font(.caption2)

                            if syncStatus.legacyPhotos > 0 {
                                Text("\(syncStatus.legacyPhotos) legacy photos")
                                    .font(.caption2)
                                    .foregroundColor(.orange)
                            }
                        }
                        .padding(.vertical, 4)
                    }

                    // Legacy photo claiming UI
                    if let syncStatus = viewModel.syncStatus, syncStatus.needsLegacyClaim {
                        Button {
                            Task {
                                await viewModel.claimLegacyPhotos()
                            }
                        } label: {
                            HStack {
                                Image(systemName: "square.and.arrow.down.on.square")
                                Text("Claim \(syncStatus.legacyPhotos) Legacy Photos")
                                Spacer()
                                if viewModel.isClaiming {
                                    ProgressView()
                                        .scaleEffect(0.8)
                                }
                            }
                        }
                        .disabled(viewModel.isClaiming)
                    }
                }

                Section {
                    Toggle(isOn: $viewModel.autoCleanupSyncedPhotos) {
                        VStack(alignment: .leading) {
                            Text("Auto-Cleanup Synced Photos")
                            Text("Automatically remove photos from device after they're synced")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    if viewModel.autoCleanupSyncedPhotos {
                        Stepper(value: $viewModel.autoCleanupAfterDays, in: 1...365, step: 1) {
                            VStack(alignment: .leading) {
                                Text("Keep Photos For")
                                Text("\(viewModel.autoCleanupAfterDays) days after sync")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                } header: {
                    Text("Photo Cleanup")
                } footer: {
                    if viewModel.autoCleanupSyncedPhotos {
                        Text("Photos older than \(viewModel.autoCleanupAfterDays) days that are synced to the server will be automatically removed from your device. They'll be moved to Recently Deleted where they'll be permanently deleted after 30 days.")
                    } else {
                        Text("Enable to automatically free up space by removing synced photos from your device")
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
