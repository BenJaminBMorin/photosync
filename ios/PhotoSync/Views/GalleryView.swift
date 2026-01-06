import SwiftUI
import Photos
import CoreData

struct GalleryView: View {
    @StateObject private var viewModel = GalleryViewModel()
    @State private var showCollections = false

    private let columns = [
        GridItem(.flexible(), spacing: 2),
        GridItem(.flexible(), spacing: 2),
        GridItem(.flexible(), spacing: 2)
    ]

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
            .navigationTitle("PhotoSync")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        showCollections = true
                    } label: {
                        Image(systemName: "square.grid.2x2")
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack {
                        Text("\(viewModel.syncedCount)/\(viewModel.photos.count)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
            .sheet(isPresented: $showCollections) {
                CollectionsView()
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
    }

    private var notConfiguredView: some View {
        VStack(spacing: 16) {
            Image(systemName: "gear")
                .font(.system(size: 64))
                .foregroundColor(.accentColor)

            Text("Server not configured")
                .font(.headline)

            Text("Please configure your server URL and API key in Settings")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)
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
                            LazyVGrid(columns: columns, spacing: 2) {
                                ForEach(group.photos) { photoState in
                                    PhotoGridItem(
                                        photoState: photoState,
                                        onTap: { viewModel.toggleSelection(for: photoState.id) },
                                        onIgnoreTap: { viewModel.toggleIgnore(for: photoState.id) }
                                    )
                                }
                            }
                            .padding(.horizontal, 2)
                            .padding(.bottom, 16)
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
        HStack {
            Button {
                viewModel.toggleUnsyncedFilter()
            } label: {
                HStack {
                    Image(systemName: viewModel.showUnsyncedOnly ? "checkmark.circle.fill" : "circle")
                    Text("Unsynced only")
                }
                .font(.subheadline)
            }
            .buttonStyle(.bordered)

            Button {
                viewModel.toggleIgnoredFilter()
            } label: {
                HStack {
                    Image(systemName: viewModel.showIgnoredPhotos ? "checkmark.circle.fill" : "circle")
                    Text("Show ignored")
                }
                .font(.subheadline)
            }
            .buttonStyle(.bordered)

            Spacer()

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
        .padding(.horizontal)
        .padding(.vertical, 8)
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
