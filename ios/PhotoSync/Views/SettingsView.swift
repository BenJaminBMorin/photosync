import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @State private var showAPIKey = false
    @State private var showChangePassword = false
    @State private var showLogoutConfirm = false
    @State private var showLogin = false
    @State private var showAdvancedConfig = false

    var body: some View {
        NavigationStack {
            Form {
                // Server & Account Section - all authentication-related UI together
                Section {
                    // Server URL input
                    HStack {
                        Image(systemName: "server.rack")
                            .foregroundColor(.blue)
                            .frame(width: 24)
                        TextField("Server URL", text: $viewModel.serverURL)
                            .textContentType(.URL)
                            .keyboardType(.URL)
                            .autocapitalization(.none)
                            .autocorrectionDisabled()
                    }

                    if viewModel.isConfigured {
                        // Signed in state
                        HStack {
                            Image(systemName: "person.circle.fill")
                                .foregroundColor(.green)
                                .frame(width: 24)
                            VStack(alignment: .leading) {
                                Text(AppSettings.userEmail ?? "Unknown")
                                    .font(.body)
                                Text("Signed In")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                        }

                        Button {
                            Task {
                                await viewModel.testConnection()
                            }
                        } label: {
                            HStack {
                                Image(systemName: "network")
                                    .frame(width: 24)
                                if viewModel.isTesting {
                                    Text("Testing...")
                                    Spacer()
                                    ProgressView()
                                        .scaleEffect(0.8)
                                } else {
                                    Text("Test Connection")
                                }
                            }
                        }
                        .disabled(viewModel.isTesting)

                        if let result = viewModel.testResult {
                            testResultView(result)
                        }

                        // Security options when signed in
                        Button {
                            showChangePassword = true
                        } label: {
                            HStack {
                                Image(systemName: "key.fill")
                                    .foregroundColor(.blue)
                                    .frame(width: 24)
                                Text("Change Password")
                                    .foregroundColor(.primary)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .foregroundColor(.gray)
                                    .font(.caption)
                            }
                        }

                        Button(role: .destructive) {
                            showLogoutConfirm = true
                        } label: {
                            HStack {
                                Image(systemName: "rectangle.portrait.and.arrow.right")
                                    .frame(width: 24)
                                Text("Sign Out")
                            }
                        }
                    } else if !viewModel.serverURL.isEmpty && viewModel.needsWebSetup {
                        // Server needs initial setup via web UI
                        HStack {
                            Image(systemName: "wrench.and.screwdriver")
                                .foregroundColor(.blue)
                                .frame(width: 24)
                            VStack(alignment: .leading) {
                                Text("Server Setup Required")
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                Text("Complete initial setup in web browser")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }

                        Button {
                            if let url = URL(string: AppSettings.normalizedServerURL) {
                                UIApplication.shared.open(url)
                            }
                        } label: {
                            HStack {
                                Image(systemName: "safari")
                                    .frame(width: 24)
                                Text("Open Server in Browser")
                                Spacer()
                                Image(systemName: "arrow.up.right.square")
                                    .font(.caption)
                            }
                        }
                        .buttonStyle(.borderedProminent)
                    } else if !viewModel.serverURL.isEmpty {
                        // Server URL set but not authenticated
                        if viewModel.isCheckingSetup {
                            HStack {
                                ProgressView()
                                    .scaleEffect(0.8)
                                Text("Checking server status...")
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                        } else {
                            HStack {
                                Image(systemName: "person.circle")
                                    .foregroundColor(.orange)
                                    .frame(width: 24)
                                VStack(alignment: .leading) {
                                    Text("Not Signed In")
                                        .font(.subheadline)
                                        .fontWeight(.medium)
                                    Text("Sign in to sync photos")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                            }

                            Button {
                                showLogin = true
                            } label: {
                                HStack {
                                    Image(systemName: "person.fill")
                                        .frame(width: 24)
                                    Text("Sign In to Server")
                                    Spacer()
                                    Image(systemName: "chevron.right")
                                        .font(.caption)
                                        .foregroundColor(.gray)
                                }
                            }
                        }
                    } else {
                        // No server configured
                        HStack {
                            Image(systemName: "exclamationmark.circle")
                                .foregroundColor(.gray)
                                .frame(width: 24)
                            Text("Enter server URL to get started")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                        }
                    }

                    // Advanced Configuration (manual API key) - collapsed by default
                    DisclosureGroup("Advanced", isExpanded: $showAdvancedConfig) {
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

                        Text("Manual API key entry for advanced users")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                } header: {
                    Text("Server & Account")
                } footer: {
                    if viewModel.serverURL.isEmpty {
                        Text("Enter your PhotoSync server address (e.g., https://photos.example.com)")
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
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Image(systemName: "clock.arrow.2.circlepath")
                                .foregroundColor(.blue)
                            Text("Background Sync Status")
                                .font(.headline)
                        }

                        Divider()

                        // Background refresh status
                        HStack {
                            Text("Background Refresh")
                                .font(.subheadline)
                            Spacer()
                            Text(viewModel.backgroundRefreshStatus)
                                .font(.caption)
                                .foregroundColor(viewModel.isBackgroundRefreshAvailable ? .green : .orange)
                        }

                        // Last background sync
                        if let lastSync = viewModel.lastBackgroundSync {
                            HStack {
                                Text("Last Background Sync")
                                    .font(.subheadline)
                                Spacer()
                                Text(viewModel.formatDate(lastSync))
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }

                        // Background sync count
                        HStack {
                            Text("Background Syncs")
                                .font(.subheadline)
                            Spacer()
                            Text("\(viewModel.backgroundSyncCount)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 4)
                } header: {
                    Text("Background Processing")
                } footer: {
                    if !viewModel.isBackgroundRefreshAvailable {
                        Text("Background refresh is disabled. Enable it in Settings > PhotoSync > Background App Refresh to allow automatic syncing when the app is closed.")
                    } else if viewModel.autoSync {
                        Text("Photos will automatically sync in the background when new photos are detected. Background tasks run every 15-30 minutes when conditions are optimal (wifi, battery).")
                    } else {
                        Text("Enable Auto-Sync to allow background processing of new photos.")
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
            .sheet(isPresented: $showLogin) {
                LoginView()
            }
            .sheet(isPresented: $showChangePassword) {
                ChangePasswordView()
            }
            .confirmationDialog(
                "Sign Out",
                isPresented: $showLogoutConfirm,
                titleVisibility: .visible
            ) {
                Button("Sign Out", role: .destructive) {
                    signOut()
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Are you sure you want to sign out? You'll need to log in again to sync photos.")
            }
            .task {
                // Check setup status when view appears (if not authenticated)
                if !viewModel.serverURL.isEmpty && !viewModel.isConfigured {
                    await viewModel.checkSetupStatus()
                }

                // Fetch current user info if already configured
                if viewModel.isConfigured {
                    await viewModel.fetchCurrentUser()
                }
            }
            .onChange(of: viewModel.serverURL) { _, newValue in
                // Check setup status when server URL changes
                if !newValue.isEmpty && !viewModel.isConfigured {
                    Task {
                        await viewModel.checkSetupStatus()
                    }
                }
            }
        }
    }

    private func signOut() {
        AppSettings.clearAuthentication()

        Task {
            await Logger.shared.info("User signed out")
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
