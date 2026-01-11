import SwiftUI

struct AllPhotosView: View {
    @StateObject private var viewModel = AllPhotosViewModel()
    @State private var showError = false
    @State private var showLogin = false

    var body: some View {
        NavigationStack {
            ZStack {
                if !AppSettings.isConfigured {
                    notSignedInView
                } else if viewModel.isLoading && viewModel.photos.isEmpty {
                    ProgressView("Loading all photos...")
                } else if viewModel.photos.isEmpty {
                    emptyStateView
                } else {
                    photoGridView
                }

                // Floating action button for selections
                if !viewModel.selectedPhotos.isEmpty {
                    floatingActionButton
                }
            }
            .navigationTitle("All Photos")
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
                                await viewModel.loadAllPhotos()
                            }
                        } label: {
                            Label("Refresh", systemImage: "arrow.clockwise")
                        }

                        Divider()

                        Button {
                            // Select all photos
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
            }
            .task {
                if AppSettings.isConfigured {
                    await viewModel.loadAllPhotos()
                }
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
        }
    }

    private var emptyStateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 64))
                .foregroundColor(.secondary)

            Text("No Photos")
                .font(.title2)
                .fontWeight(.semibold)

            Text("Photos from your device and cloud will appear here")
                .font(.body)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)
        }
        .padding()
    }

    private var notSignedInView: some View {
        VStack(spacing: 20) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 64))
                .foregroundColor(.blue)

            Text("Sign In to View All Photos")
                .font(.title2)
                .fontWeight(.semibold)

            Text("Connect to your PhotoSync server to see photos from both your device and the cloud")
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
            // Stats bar
            statsBar

            // Photo grid
            ScrollView {
                LazyVGrid(columns: columns, spacing: 4) {
                    ForEach(viewModel.displayedPhotos) { photo in
                        AllPhotoCard(
                            photo: photo,
                            isSelected: viewModel.selectedPhotos.contains(photo.id),
                            onTap: {
                                viewModel.toggleSelection(for: photo.id)
                            }
                        )
                        .onAppear {
                            viewModel.loadThumbnailIfNeeded(for: photo)
                        }
                    }
                }
                .padding(8)
            }
        }
    }

    private var columns: [GridItem] {
        let deviceWidth = UIScreen.main.bounds.width
        let columnCount = deviceWidth > 600 ? 5 : 3
        return Array(repeating: GridItem(.flexible(), spacing: 4), count: columnCount)
    }

    private var statsBar: some View {
        HStack(spacing: 16) {
            StatBadge(
                label: "Total",
                count: viewModel.photos.count,
                icon: "photo.on.rectangle.angled",
                color: .blue
            )

            StatBadge(
                label: "On Device",
                count: viewModel.photos.filter { $0.isLocal }.count,
                icon: "iphone",
                color: .green
            )

            StatBadge(
                label: "In Cloud",
                count: viewModel.photos.filter { $0.isServer }.count,
                icon: "cloud.fill",
                color: .orange
            )
        }
        .padding()
        .background(Color(.systemBackground))
    }

    private var floatingActionButton: some View {
        VStack {
            Spacer()
            HStack {
                Spacer()
                Menu {
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

struct AllPhotoCard: View {
    let photo: CombinedPhoto
    let isSelected: Bool
    let onTap: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            ZStack {
                if let thumbnail = photo.thumbnail {
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

                // Source badge
                VStack {
                    HStack {
                        ZStack {
                            Circle()
                                .fill(photo.isLocal ? .green : .blue)
                                .frame(width: 20, height: 20)

                            Image(systemName: photo.isLocal ? "iphone" : "cloud.fill")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundColor(.white)
                        }
                        .padding(4)
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
            }
            .frame(width: 100, height: 100)
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .onTapGesture {
                onTap()
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(photo.filename)
                    .font(.caption2)
                    .lineLimit(1)
                    .truncationMode(.middle)
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
    AllPhotosView()
}
