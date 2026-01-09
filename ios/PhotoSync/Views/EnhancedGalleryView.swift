import SwiftUI
import Photos
import CoreData

/// Enhanced gallery view with better design and UX
struct EnhancedGalleryView: View {
    @StateObject private var viewModel = GalleryViewModel()
    @State private var showCollections = false
    @State private var showDeleteConfirmation = false
    @State private var photoToDelete: String?
    @State private var activeFilter: PhotoFilter = .all

    // Adaptive grid columns based on device
    private var columns: [GridItem] {
        let deviceWidth = UIScreen.main.bounds.width
        let columnCount = deviceWidth > 600 ? 5 : 3  // iPad vs iPhone
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
                    loadingView
                } else if viewModel.photos.isEmpty {
                    emptyStateView
                } else {
                    photoGridView
                }

                // Sync progress overlay
                if viewModel.isSyncing {
                    SyncProgressView(
                        progress: viewModel.syncProgress,
                        onCancel: { viewModel.cancelSync() }
                    )
                }

                // Floating action button for selections
                if viewModel.selectedCount > 0 {
                    FloatingActionButton(
                        selectedCount: viewModel.selectedCount,
                        onSyncTap: {
                            viewModel.syncSelected()
                        },
                        onHideTap: {
                            viewModel.ignoreSelected()
                        },
                        onDeleteTap: {
                            // Delete all selected photos
                            // For now, just show confirmation for first photo
                            if let firstSelected = viewModel.photos.first(where: { $0.isSelected }) {
                                photoToDelete = firstSelected.id
                                showDeleteConfirmation = true
                            }
                        },
                        onClearSelection: {
                            viewModel.clearSelection()
                        }
                    )
                }
            }
            .navigationTitle("On Device")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        showCollections = true
                    } label: {
                        HStack(spacing: 6) {
                            Image(systemName: "folder")
                                .font(.body.weight(.medium))
                            Text("Albums")
                                .font(.body)
                        }
                        .foregroundColor(.blue)
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button {
                            Task {
                                await viewModel.loadPhotos()
                            }
                        } label: {
                            Label("Refresh", systemImage: "arrow.clockwise")
                        }

                        Button {
                            viewModel.selectAll()
                        } label: {
                            Label("Select All Unsynced", systemImage: "checkmark.circle")
                        }

                        Divider()

                        Button {
                            viewModel.resetFilters()
                        } label: {
                            Label("Reset Filters", systemImage: "arrow.counterclockwise")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.body.weight(.medium))
                    }
                }
            }
            .sheet(isPresented: $showCollections) {
                CollectionsView()
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
                        Text("This photo is safely backed up. It will be moved to Recently Deleted.")
                    } else {
                        Text("⚠️ This photo has NOT been synced. If deleted, it will be lost.")
                    }
                } else {
                    Text("This photo will be moved to Recently Deleted.")
                }
            }
            .alert("Error", isPresented: .constant(viewModel.error != nil)) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.error ?? "")
            }
        }
        .task {
            await viewModel.requestAuthorization()
        }
        .refreshable {
            await viewModel.loadPhotos()
        }
    }

    // MARK: - Main Content

    private var photoGridView: some View {
        VStack(spacing: 0) {
            // Stats card
            SyncStatsCard(
                totalPhotos: filteredPhotos.count,
                syncedCount: filteredPhotos.filter { $0.syncState == .synced }.count,
                unsyncedCount: filteredPhotos.filter { $0.syncState != .synced && $0.syncState != .ignored }.count,
                ignoredCount: filteredPhotos.filter { $0.syncState == .ignored }.count,
                activeFilter: $activeFilter
            )

            // Photo grid with enhanced items
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0, pinnedViews: [.sectionHeaders]) {
                    ForEach(groupedFilteredPhotos) { group in
                        Section {
                            LazyVGrid(columns: columns, spacing: 4) {
                                ForEach(group.photos) { photoState in
                                    EnhancedPhotoGridItem(
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
                            EnhancedSectionHeader(
                                title: group.displayTitle,
                                syncedCount: group.syncedCount,
                                totalCount: group.totalCount
                            )
                        }
                    }

                    // Bottom padding to account for FAB
                    Color.clear
                        .frame(height: viewModel.selectedCount > 0 ? 100 : 20)
                }
                .padding(.top, 4)
            }
        }
    }

    // MARK: - Empty States

    private var notConfiguredView: some View {
        EmptyStateView(
            icon: "gear",
            title: "Server Not Configured",
            message: "Please configure your server URL and API key in Settings",
            iconColor: .blue
        )
    }

    private var requestAccessView: some View {
        EmptyStateView(
            icon: "photo.on.rectangle",
            title: "Photo Access Required",
            message: "PhotoSync needs access to your photos to sync them to your server",
            iconColor: .blue,
            action: {
                Button("Grant Access") {
                    Task {
                        await viewModel.requestAuthorization()
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        )
    }

    private var accessDeniedView: some View {
        EmptyStateView(
            icon: "photo.on.rectangle.angled",
            title: "Photo Access Denied",
            message: "Please enable photo access in Settings to use PhotoSync",
            iconColor: .red,
            action: {
                Button("Open Settings") {
                    if let url = URL(string: UIApplication.openSettingsURLString) {
                        UIApplication.shared.open(url)
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        )
    }

    private var loadingView: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.5)
                .tint(.blue)

            Text("Loading photos...")
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
    }

    private var emptyStateView: some View {
        EmptyStateView(
            icon: "photo.stack",
            title: "No Photos Found",
            message: "Take some photos to get started!",
            iconColor: .secondary
        )
    }

    // MARK: - Filtering

    private var filteredPhotos: [PhotoWithState] {
        let photos = viewModel.displayedPhotos

        switch activeFilter {
        case .all:
            return photos
        case .unsynced:
            return photos.filter { $0.syncState != .synced && $0.syncState != .ignored }
        case .today:
            let today = Calendar.current.startOfDay(for: Date())
            return photos.filter { $0.photo.creationDate >= today }
        case .thisWeek:
            let weekAgo = Calendar.current.date(byAdding: .day, value: -7, to: Date()) ?? Date()
            return photos.filter { $0.photo.creationDate >= weekAgo }
        case .hidden:
            return photos.filter { $0.syncState == .ignored }
        }
    }

    private var groupedFilteredPhotos: [PhotoGroup] {
        let grouped = Dictionary(grouping: filteredPhotos) { photo -> String in
            let components = Calendar.current.dateComponents([.year, .month], from: photo.photo.creationDate)
            return String(format: "%04d-%02d", components.year ?? 0, components.month ?? 0)
        }

        return grouped.map { key, photos in
            let components = key.split(separator: "-")
            let year = Int(components[0]) ?? 0
            let month = Int(components[1]) ?? 0
            return PhotoGroup(id: key, year: year, month: month, photos: photos)
        }
        .sorted { $0.year == $1.year ? $0.month > $1.month : $0.year > $1.year }
    }
}

// MARK: - Supporting Views

private struct EnhancedSectionHeader: View {
    let title: String
    let syncedCount: Int
    let totalCount: Int

    var body: some View {
        HStack(spacing: 12) {
            Text(title)
                .font(.title3.bold())

            Spacer()

            // Sync progress indicator
            HStack(spacing: 6) {
                // Mini progress ring
                ZStack {
                    Circle()
                        .stroke(Color.secondary.opacity(0.2), lineWidth: 2)
                        .frame(width: 20, height: 20)

                    Circle()
                        .trim(from: 0, to: totalCount > 0 ? Double(syncedCount) / Double(totalCount) : 0)
                        .stroke(syncColor, lineWidth: 2)
                        .frame(width: 20, height: 20)
                        .rotationEffect(.degrees(-90))
                }

                Text("\(syncedCount)/\(totalCount)")
                    .font(.subheadline.monospacedDigit())
                    .foregroundColor(.secondary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(
            Color(.systemBackground)
                .overlay(
                    Rectangle()
                        .fill(Color.secondary.opacity(0.1))
                        .frame(height: 1),
                    alignment: .bottom
                )
        )
    }

    private var syncColor: Color {
        guard totalCount > 0 else { return .gray }
        let percentage = Double(syncedCount) / Double(totalCount)
        if percentage == 1.0 {
            return .green
        } else if percentage > 0.5 {
            return .blue
        } else {
            return .orange
        }
    }
}

private struct EmptyStateView: View {
    let icon: String
    let title: String
    let message: String
    let iconColor: Color
    var action: (() -> any View)? = nil

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: icon)
                .font(.system(size: 72, weight: .thin))
                .foregroundColor(iconColor)

            VStack(spacing: 8) {
                Text(title)
                    .font(.title2.weight(.semibold))

                Text(message)
                    .font(.body)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 32)
            }

            if let action = action {
                AnyView(action())
            }
        }
        .padding()
    }
}

// MARK: - Preview

#Preview {
    EnhancedGalleryView()
        .environment(\.managedObjectContext, PersistenceController.preview.container.viewContext)
}
