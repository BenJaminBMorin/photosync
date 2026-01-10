import SwiftUI
import Photos
import CoreData

struct GalleryView: View {
    @StateObject private var viewModel = GalleryViewModel()
    @State private var showCollections = false
    @State private var showFilterOptions = false
    @State private var showDeleteConfirmation = false
    @State private var photoToDelete: String?
    @State private var showLogin = false

    // Adaptive columns based on device size
    private var columns: [GridItem] {
        let deviceWidth = UIScreen.main.bounds.width
        let columnCount = deviceWidth > 600 ? 5 : 3  // iPad gets 5, iPhone gets 3
        return Array(repeating: GridItem(.flexible(), spacing: 4), count: columnCount)
    }

    var body: some View {
        NavigationStack {
            ZStack {
                if !viewModel.isConfigured {
                    notConfiguredView
                } else if viewModel.authorizationStatus == .notDetermined {
                    requestAccessView
                } else if viewModel.authorizationStatus == .denied || viewModel.authorizationStatus == .restricted {
                    accessDeniedView
                } else if viewModel.isLoading {
                    ProgressView("Loading photos...")
                } else if viewModel.photos.isEmpty {
                    Text("No photos found")
                        .foregroundColor(.secondary)
                } else {
                    photoGridView
                }

                if viewModel.isSyncing {
                    SyncProgressView(
                        progress: viewModel.syncProgress,
                        onCancel: { viewModel.cancelSync() }
                    )
                }
            }
            .navigationTitle("On Device")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        showCollections = true
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "folder")
                            Text("Albums")
                                .font(.body)
                        }
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack(spacing: 8) {
                        // Sync status badge
                        HStack(spacing: 4) {
                            Image(systemName: "icloud.and.arrow.up")
                                .font(.caption)
                            Text("\(viewModel.syncedCount)/\(viewModel.photos.count)")
                                .font(.caption.monospacedDigit())
                        }
                        .foregroundColor(viewModel.unsyncedCount > 0 ? .orange : .green)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.secondary.opacity(0.1))
                        .clipShape(Capsule())
                    }
                }
            }
            .sheet(isPresented: $showCollections) {
                CollectionsView()
            }
            .sheet(isPresented: $showFilterOptions) {
                FilterOptionsView(viewModel: viewModel)
            }
            .alert("Error", isPresented: .constant(viewModel.error != nil)) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.error ?? "")
            }
            .alert("Delete Photo", isPresented: $showDeleteConfirmation) {
                Button("Cancel", role: .cancel) {
                    photoToDelete = nil
                }
                Button("Delete", role: .destructive) {
                    if let photoId = photoToDelete {
                        Task {
                            do {
                                try await viewModel.deletePhoto(for: photoId)
                            } catch {
                                viewModel.error = "Failed to delete photo: \(error.localizedDescription)"
                            }
                        }
                    }
                    photoToDelete = nil
                }
            } message: {
                if let photoId = photoToDelete,
                   let photo = viewModel.photos.first(where: { $0.id == photoId }) {
                    if photo.syncState == .synced {
                        Text("This photo is safely backed up on your server. It will be moved to Recently Deleted and permanently removed after 30 days.")
                    } else {
                        Text("⚠️ WARNING: This photo has NOT been synced to your server yet. If you delete it now, it will be lost forever after 30 days in Recently Deleted. Consider syncing it first!")
                    }
                } else {
                    Text("This photo will be moved to your Recently Deleted album where it will be permanently deleted after 30 days.")
                }
            }
        }
        .task {
            await viewModel.requestAuthorization()
        }
    }

    private var notConfiguredView: some View {
        VStack(spacing: 20) {
            Image(systemName: "icloud.and.arrow.up")
                .font(.system(size: 64))
                .foregroundColor(.accentColor)

            Text("Welcome to PhotoSync")
                .font(.title2)
                .fontWeight(.bold)

            Text("Sign in to start syncing your photos to your server")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            VStack(spacing: 12) {
                Button {
                    showLogin = true
                } label: {
                    HStack {
                        Image(systemName: "person.fill")
                        Text("Sign In with Password")
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.large)

                Text("or configure manually in Settings")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal, 32)
        }
        .sheet(isPresented: $showLogin) {
            LoginView()
        }
    }

    private var requestAccessView: some View {
        VStack(spacing: 16) {
            Image(systemName: "photo.on.rectangle")
                .font(.system(size: 64))
                .foregroundColor(.accentColor)

            Text("Photo Access Required")
                .font(.headline)

            Text("PhotoSync needs access to your photos to sync them to your server")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Button("Grant Access") {
                Task {
                    await viewModel.requestAuthorization()
                }
            }
            .buttonStyle(.borderedProminent)
        }
    }

    private var accessDeniedView: some View {
        VStack(spacing: 16) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 64))
                .foregroundColor(.red)

            Text("Photo Access Denied")
                .font(.headline)

            Text("Please enable photo access in Settings to use PhotoSync")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Button("Open Settings") {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            .buttonStyle(.borderedProminent)
        }
    }

    private var photoGridView: some View {
        VStack(spacing: 0) {
            // Filter bar
            filterBar

            // Photo grid grouped by month
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0, pinnedViews: [.sectionHeaders]) {
                    ForEach(viewModel.groupedPhotos) { group in
                        Section {
                            LazyVGrid(columns: columns, spacing: 4) {
                                ForEach(group.photos) { photoState in
                                    PhotoGridItem(
                                        photoState: photoState,
                                        onTap: { viewModel.toggleSelection(for: photoState.id) },
                                        onIgnoreTap: { viewModel.toggleIgnore(for: photoState.id) },
                                        onDeleteTap: {
                                            photoToDelete = photoState.id
                                            showDeleteConfirmation = true
                                        }
                                    )
                                }
                            }
                            .padding(.horizontal, 4)
                            .padding(.bottom, 20)
                        } header: {
                            HStack {
                                Text(group.displayTitle)
                                    .font(.title3.bold())
                                Spacer()
                                Text("\(group.syncedCount)/\(group.totalCount) synced")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            .padding(.horizontal)
                            .padding(.vertical, 8)
                            .background(Color(.systemBackground))
                        }
                    }
                }
                .padding(.top, 2)
            }

            // Bottom bar with selection controls
            if viewModel.selectedCount > 0 {
                selectionBar
            }
        }
    }

    private var filterBar: some View {
        HStack(spacing: 8) {
            // Scrollable filter chips area
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    // Filter button with indicator
                    Button {
                        showFilterOptions = true
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "line.3.horizontal.decrease.circle" + (hasActiveFilters ? ".fill" : ""))
                            Text("Filters")
                            if hasActiveFilters {
                                Text("(\(activeFilterCount))")
                                    .font(.caption2)
                            }
                        }
                        .font(.subheadline)
                    }
                    .buttonStyle(.borderedProminent)

                    // Quick filter chips
                    if viewModel.showUnsyncedOnly {
                        filterChip(text: "Unsynced", icon: "icloud.slash") {
                            viewModel.showUnsyncedOnly = false
                        }
                    }

                    if viewModel.showIgnoredPhotos {
                        filterChip(text: "Ignored", icon: "eye.slash") {
                            viewModel.showIgnoredPhotos = false
                        }
                    }

                    if viewModel.enableDateFilter {
                        filterChip(text: "Date", icon: "calendar") {
                            viewModel.enableDateFilter = false
                        }
                    }
                }
                .padding(.horizontal)
            }

            // Fixed action buttons on the right
            HStack(spacing: 8) {
                Button {
                    viewModel.selectAll()
                } label: {
                    Image(systemName: "checkmark.circle")
                }

                Button {
                    Task { await viewModel.loadPhotos() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
            .padding(.trailing)
        }
        .padding(.vertical, 8)
        .background(Color(.systemBackground))
    }

    private func filterChip(text: String, icon: String, onRemove: @escaping () -> Void) -> some View {
        Button {
            onRemove()
        } label: {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.caption)
                Text(text)
                    .font(.caption)
                Image(systemName: "xmark.circle.fill")
                    .font(.caption2)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(Color.accentColor.opacity(0.2))
            .foregroundColor(.accentColor)
            .cornerRadius(12)
        }
    }

    private var hasActiveFilters: Bool {
        !viewModel.showUnsyncedOnly ||
        viewModel.showIgnoredPhotos ||
        viewModel.showServerOnlyPhotos ||
        viewModel.enableDateFilter
    }

    private var activeFilterCount: Int {
        var count = 0
        if !viewModel.showUnsyncedOnly { count += 1 } // Showing all is a filter
        if viewModel.showIgnoredPhotos { count += 1 }
        if viewModel.showServerOnlyPhotos { count += 1 }
        if viewModel.enableDateFilter { count += 1 }
        return count
    }

    private var selectionBar: some View {
        HStack(spacing: 12) {
            Text("\(viewModel.selectedCount) selected")
                .font(.subheadline)

            Spacer()

            Button {
                viewModel.clearSelection()
            } label: {
                Image(systemName: "xmark.circle")
            }
            .buttonStyle(.bordered)

            if viewModel.selectedIgnoredCount > 0 {
                Button {
                    viewModel.unignoreSelected()
                } label: {
                    Image(systemName: "eye.fill")
                }
                .buttonStyle(.bordered)
            }

            if viewModel.selectedNonIgnoredCount > 0 {
                Button {
                    viewModel.ignoreSelected()
                } label: {
                    Image(systemName: "eye.slash.fill")
                }
                .buttonStyle(.bordered)
            }

            Button {
                viewModel.syncSelected()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "icloud.and.arrow.up")
                    Text("Sync")
                }
            }
            .buttonStyle(.borderedProminent)
            .disabled(!viewModel.isConfigured)
        }
        .padding()
        .background(Color(.systemBackground))
        .shadow(radius: 2)
    }
}

#Preview {
    GalleryView()
        .environment(\.managedObjectContext, PersistenceController.preview.container.viewContext)
}
