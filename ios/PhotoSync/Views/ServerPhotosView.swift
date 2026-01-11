import SwiftUI

struct ServerPhotosView: View {
    @StateObject private var viewModel = ServerPhotosViewModel()
    @State private var selectedPhoto: ServerPhoto?
    @State private var showRestoreConfirmation = false
    @State private var showShareSheet = false
    @State private var shareItems: [Any] = []
    @State private var showDeleteConfirmation = false
    @State private var showCollectionsSheet = false
    @State private var showCreateCollectionDialog = false
    @State private var newCollectionName = ""
    @State private var showDateFilter = false
    @State private var showLogin = false

    var body: some View {
        NavigationStack {
            ZStack {
                if !AppSettings.isConfigured {
                    notSignedInView
                } else if viewModel.isLoading {
                    ProgressView("Loading server photos...")
                } else if viewModel.serverPhotos.isEmpty {
                    emptyStateView
                } else {
                    photoGridView
                }

                // Floating action button for selections
                if !viewModel.selectedPhotos.isEmpty {
                    floatingActionButton
                }
            }
            .navigationTitle("In Cloud")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    if !viewModel.selectedPhotos.isEmpty {
                        Button("Cancel") {
                            viewModel.clearSelection()
                        }
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button {
                            Task {
                                await viewModel.loadServerPhotos()
                            }
                        } label: {
                            Label("Refresh", systemImage: "arrow.clockwise")
                        }

                        Divider()

                        Button {
                            viewModel.showNotOnDeviceOnly.toggle()
                        } label: {
                            Label(
                                viewModel.showNotOnDeviceOnly ? "Show All" : "Show Not on Device Only",
                                systemImage: viewModel.showNotOnDeviceOnly ? "iphone" : "iphone.slash"
                            )
                        }

                        Divider()

                        Button {
                            // Select all visible photos
                            for photo in viewModel.displayedPhotos {
                                viewModel.selectedPhotos.insert(photo.id)
                            }
                        } label: {
                            Label("Select All", systemImage: "checkmark.circle")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.body.weight(.medium))
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        Task {
                            await viewModel.loadCollections()
                        }
                        showCollectionsSheet = true
                    } label: {
                        Image(systemName: "folder.fill")
                            .font(.body.weight(.medium))
                    }
                }
            }
            .task(id: AppSettings.isConfigured) {
                if AppSettings.isConfigured {
                    await viewModel.loadServerPhotos()
                }
            }
            .alert("Restore Photo", isPresented: $showRestoreConfirmation, presenting: selectedPhoto) { photo in
                Button("Cancel", role: .cancel) {
                    selectedPhoto = nil
                }
                Button("Restore") {
                    Task {
                        await viewModel.restorePhoto(photo)
                        selectedPhoto = nil
                    }
                }
            } message: { photo in
                Text("Download \(photo.originalFilename) to your photo library?")
            }
            .alert("Error", isPresented: .constant(viewModel.error != nil)) {
                Button("OK") {
                    viewModel.clearError()
                }
            } message: {
                if let error = viewModel.error {
                    Text(error)
                }
            }
            .alert("Delete Photos", isPresented: $showDeleteConfirmation) {
                Button("Cancel", role: .cancel) { }
                Button("Delete", role: .destructive) {
                    Task {
                        await viewModel.deleteSelectedPhotos()
                    }
                }
            } message: {
                Text("Delete \(viewModel.selectedPhotos.count) photo(s) from the server? This action cannot be undone.")
            }
            .sheet(isPresented: $showCollectionsSheet) {
                collectionsSheet
            }
        }
    }

    private var collectionsSheet: some View {
        NavigationStack {
            Group {
                if viewModel.isLoadingCollections {
                    ProgressView("Loading collections...")
                } else if viewModel.collections.isEmpty {
                    VStack(spacing: 16) {
                        Image(systemName: "folder.badge.plus")
                            .font(.system(size: 48))
                            .foregroundColor(.secondary)

                        Text("No Collections Yet")
                            .font(.title2)
                            .fontWeight(.semibold)

                        Text("Create a collection to organize your photos")
                            .font(.body)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)

                        Button {
                            showCreateCollectionDialog = true
                        } label: {
                            Label("Create Collection", systemImage: "plus.circle.fill")
                                .font(.headline)
                        }
                        .buttonStyle(.borderedProminent)
                        .padding(.top)
                    }
                    .padding()
                } else {
                    List {
                        ForEach(viewModel.collections) { collection in
                            Button {
                                Task {
                                    await viewModel.addSelectedPhotosToCollection(collection)
                                    showCollectionsSheet = false
                                }
                            } label: {
                                HStack {
                                    Image(systemName: "folder.fill")
                                        .foregroundColor(.blue)

                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(collection.name)
                                            .font(.headline)
                                            .foregroundColor(.primary)

                                        Text("\(collection.photoCount) photos")
                                            .font(.caption)
                                            .foregroundColor(.secondary)
                                    }

                                    Spacer()

                                    Image(systemName: "chevron.right")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                .padding(.vertical, 4)
                            }
                            .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                                Button(role: .destructive) {
                                    Task {
                                        await viewModel.deleteCollection(collection)
                                    }
                                } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("Add to Collection")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") {
                        showCollectionsSheet = false
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showCreateCollectionDialog = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .alert("Create Collection", isPresented: $showCreateCollectionDialog) {
                TextField("Collection Name", text: $newCollectionName)
                Button("Cancel", role: .cancel) {
                    newCollectionName = ""
                }
                Button("Create") {
                    Task {
                        let success = await viewModel.createCollection(name: newCollectionName)
                        if success {
                            newCollectionName = ""
                        }
                    }
                }
            } message: {
                Text("Enter a name for the new collection")
            }
        }
    }

    private var emptyStateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "checkmark.circle")
                .font(.system(size: 64))
                .foregroundColor(.green)

            Text("All Synced!")
                .font(.title2)
                .fontWeight(.semibold)

            Text("All photos on the server are also on your device")
                .font(.body)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)
        }
        .padding()
    }

    private var notSignedInView: some View {
        VStack(spacing: 20) {
            Image(systemName: "cloud.fill")
                .font(.system(size: 64))
                .foregroundColor(.blue)

            Text("Sign In to View Cloud Photos")
                .font(.title2)
                .fontWeight(.semibold)

            Text("Connect to your PhotoSync server to see photos stored in the cloud")
                .font(.body)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Button {
                showLogin = true
            } label: {
                Label("Sign In", systemImage: "person.fill")
                    .font(.headline)
            }
            .buttonStyle(.borderedProminent)
            .padding(.top)
        }
        .padding()
        .sheet(isPresented: $showLogin) {
            LoginView()
        }
    }

    private var photoGridView: some View {
        VStack(spacing: 0) {
            // Stats and filter bar
            statsBar

            // Photo grid
            ScrollView {
                LazyVGrid(columns: [
                    GridItem(.adaptive(minimum: 100), spacing: 4)
                ], spacing: 4) {
                    ForEach(viewModel.displayedPhotos) { photoWithThumbnail in
                        ServerPhotoCard(
                            photo: photoWithThumbnail.photo,
                            thumbnail: photoWithThumbnail.thumbnail,
                            isSelected: viewModel.selectedPhotos.contains(photoWithThumbnail.id),
                            onTap: {
                                viewModel.toggleSelection(for: photoWithThumbnail.id)
                            },
                            onRestore: {
                                selectedPhoto = photoWithThumbnail.photo
                                showRestoreConfirmation = true
                            }
                        )
                    .onAppear {
                        // Load thumbnail when photo appears
                        viewModel.loadThumbnailIfNeeded(for: photoWithThumbnail)

                        // Load more photos if needed
                        Task {
                            await viewModel.loadMoreIfNeeded(currentPhoto: photoWithThumbnail)
                        }
                    }
                }

                // Loading indicator at bottom
                if viewModel.isLoadingMore {
                    VStack {
                        ProgressView()
                            .scaleEffect(1.2)
                            .padding()
                        Text("Loading more...")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                }
            }
            .padding(8)
            }
        }
    }

    private var statsBar: some View {
        VStack(spacing: 12) {
            // Photo count stats
            HStack(spacing: 16) {
                StatBadge(
                    label: "Total",
                    count: viewModel.serverPhotos.count,
                    icon: "cloud.fill",
                    color: .blue
                )

                StatBadge(
                    label: "On Device",
                    count: viewModel.onDeviceCount,
                    icon: "iphone",
                    color: .green
                )

                StatBadge(
                    label: "Not on Device",
                    count: viewModel.notOnDeviceCount,
                    icon: "iphone.slash",
                    color: .orange
                )
            }
            .padding(.horizontal)

            // Filter toggles
            VStack(spacing: 8) {
                // Device filter
                if viewModel.notOnDeviceCount > 0 {
                    Button {
                        withAnimation {
                            viewModel.showNotOnDeviceOnly.toggle()
                        }
                    } label: {
                        HStack {
                            Image(systemName: viewModel.showNotOnDeviceOnly ? "checkmark.circle.fill" : "circle")
                                .foregroundColor(viewModel.showNotOnDeviceOnly ? .blue : .secondary)
                            Text("Show only photos not on this device")
                                .font(.subheadline)
                            Spacer()
                        }
                        .padding()
                        .background(Color(.systemGroupedBackground))
                        .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                }

                // Date filter toggle
                Button {
                    withAnimation {
                        showDateFilter.toggle()
                    }
                } label: {
                    HStack {
                        Image(systemName: viewModel.enableDateFilter ? "checkmark.circle.fill" : "circle")
                            .foregroundColor(viewModel.enableDateFilter ? .blue : .secondary)
                        Text("Filter by date")
                            .font(.subheadline)
                        Spacer()
                        if viewModel.enableDateFilter {
                            Text(dateRangeText)
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        Image(systemName: showDateFilter ? "chevron.up" : "chevron.down")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding()
                    .background(Color(.systemGroupedBackground))
                    .cornerRadius(8)
                }
                .buttonStyle(.plain)

                // Date picker section (collapsible)
                if showDateFilter {
                    VStack(spacing: 12) {
                        Toggle("Enable date filter", isOn: $viewModel.enableDateFilter)
                            .tint(.blue)

                        if viewModel.enableDateFilter {
                            // Preset buttons
                            HStack(spacing: 8) {
                                datePresetButton(title: "7 days", days: 7)
                                datePresetButton(title: "30 days", days: 30)
                                datePresetButton(title: "90 days", days: 90)
                                datePresetButton(title: "1 year", days: 365)
                            }

                            Divider()

                            // Custom date pickers
                            DatePicker("From", selection: $viewModel.dateFilterStart, displayedComponents: .date)
                            DatePicker("To", selection: $viewModel.dateFilterEnd, displayedComponents: .date)
                        }
                    }
                    .padding()
                    .background(Color(.systemGroupedBackground))
                    .cornerRadius(8)
                }
            }
            .padding(.horizontal)
        }
        .padding(.vertical, 12)
        .background(Color(.systemBackground))
    }

    private var dateRangeText: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        return "\(formatter.string(from: viewModel.dateFilterStart)) - \(formatter.string(from: viewModel.dateFilterEnd))"
    }

    private func datePresetButton(title: String, days: Int) -> some View {
        Button {
            viewModel.dateFilterStart = Calendar.current.date(byAdding: .day, value: -days, to: Date()) ?? Date()
            viewModel.dateFilterEnd = Date()
        } label: {
            Text(title)
                .font(.caption)
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(Color.blue.opacity(0.1))
                .foregroundColor(.blue)
                .cornerRadius(16)
        }
    }

    private var floatingActionButton: some View {
        VStack {
            Spacer()
            HStack {
                Spacer()
                Menu {
                    Button {
                        // Download selected photos
                        Task {
                            await downloadSelectedPhotos()
                        }
                    } label: {
                        Label("Download to Device (\(viewModel.selectedPhotos.count))", systemImage: "arrow.down.circle")
                    }

                    Button {
                        // Share selected photos
                        shareSelectedPhotos()
                    } label: {
                        Label("Share", systemImage: "square.and.arrow.up")
                    }

                    Button {
                        // Add to collection
                        Task {
                            await viewModel.loadCollections()
                        }
                        showCollectionsSheet = true
                    } label: {
                        Label("Add to Collection", systemImage: "folder.badge.plus")
                    }

                    Divider()

                    Button(role: .destructive) {
                        showDeleteConfirmation = true
                    } label: {
                        Label("Delete from Server", systemImage: "trash")
                    }

                    Divider()

                    Button {
                        viewModel.clearSelection()
                    } label: {
                        Label("Clear Selection", systemImage: "xmark.circle")
                    }
                } label: {
                    ZStack {
                        Circle()
                            .fill(
                                LinearGradient(
                                    colors: [.blue, .blue.opacity(0.8)],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                )
                            )
                            .frame(width: 64, height: 64)
                            .shadow(color: .black.opacity(0.3), radius: 8, x: 0, y: 4)

                        VStack(spacing: 2) {
                            Image(systemName: "checkmark.circle")
                                .font(.system(size: 20, weight: .semibold))

                            Text("\(viewModel.selectedPhotos.count)")
                                .font(.caption2.bold())
                                .monospacedDigit()
                        }
                        .foregroundColor(.white)
                    }
                }
                .padding(.trailing, 20)
                .padding(.bottom, 20)
            }
        }
    }

    private func downloadSelectedPhotos() async {
        let selectedPhotos = viewModel.displayedPhotos.filter { viewModel.selectedPhotos.contains($0.id) }

        for photoWithThumbnail in selectedPhotos {
            await viewModel.restorePhoto(photoWithThumbnail.photo)
        }

        viewModel.clearSelection()
    }

    private func shareSelectedPhotos() {
        let selectedPhotos = viewModel.displayedPhotos.filter { viewModel.selectedPhotos.contains($0.id) }

        // For now, share URLs or file names
        let shareText = selectedPhotos.map { $0.photo.originalFilename }.joined(separator: "\n")
        shareItems = [shareText]
        showShareSheet = true
    }
}

// MARK: - Supporting Views

private struct StatBadge: View {
    let label: String
    let count: Int
    let icon: String
    let color: Color

    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundColor(color)

            Text("\(count)")
                .font(.title2.bold())
                .monospacedDigit()

            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(color.opacity(0.1))
        .cornerRadius(8)
    }
}

struct ServerPhotoCard: View {
    let photo: ServerPhoto
    let thumbnail: UIImage?
    let isSelected: Bool
    let onTap: () -> Void
    let onRestore: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            ZStack {
                if let thumbnail = thumbnail {
                    Image(uiImage: thumbnail)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(width: 100, height: 100)
                        .clipped()
                } else {
                    Rectangle()
                        .fill(Color.gray.opacity(0.3))
                        .frame(width: 100, height: 100)

                    ProgressView()
                }

                // Device status badge
                VStack {
                    HStack {
                        if photo.isOnDevice {
                            ZStack {
                                Circle()
                                    .fill(.green)
                                    .frame(width: 20, height: 20)

                                Image(systemName: "iphone")
                                    .font(.system(size: 10, weight: .bold))
                                    .foregroundColor(.white)
                            }
                            .padding(4)
                        } else {
                            ZStack {
                                Circle()
                                    .fill(.orange)
                                    .frame(width: 20, height: 20)

                                Image(systemName: "iphone.slash")
                                    .font(.system(size: 10, weight: .bold))
                                    .foregroundColor(.white)
                            }
                            .padding(4)
                        }
                        Spacer()
                    }
                    Spacer()
                }

                // Selection overlay
                if isSelected {
                    ZStack {
                        Color.black.opacity(0.3)

                        VStack {
                            HStack {
                                Spacer()
                                ZStack {
                                    Circle()
                                        .fill(.blue)
                                        .frame(width: 28, height: 28)

                                    Image(systemName: "checkmark")
                                        .font(.system(size: 14, weight: .bold))
                                        .foregroundColor(.white)
                                }
                                .padding(4)
                            }
                            Spacer()
                        }
                    }
                }

                // Restore button overlay (only show if not on device and not selected)
                if !photo.isOnDevice && !isSelected {
                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            Button {
                                onRestore()
                            } label: {
                                Image(systemName: "arrow.down.circle.fill")
                                    .font(.title2)
                                    .foregroundColor(.white)
                                    .shadow(color: .black.opacity(0.3), radius: 2)
                            }
                            .padding(8)
                        }
                    }
                }
            }
            .frame(width: 100, height: 100)
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .onTapGesture {
                onTap()
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(photo.originalFilename)
                    .font(.caption2)
                    .lineLimit(1)
                    .truncationMode(.middle)

                Text(photo.formattedFileSize)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 4)
            .padding(.vertical, 2)
        }
        .background(Color(.systemBackground))
        .cornerRadius(8)
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(
                    LinearGradient(
                        colors: isSelected ? [.blue, .blue.opacity(0.7)] : [.clear],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    ),
                    lineWidth: isSelected ? 3 : 0
                )
        )
        .shadow(radius: isSelected ? 4 : 2)
    }
}

#Preview {
    ServerPhotosView()
}
