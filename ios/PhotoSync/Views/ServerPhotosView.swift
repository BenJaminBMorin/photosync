import SwiftUI

struct ServerPhotosView: View {
    @StateObject private var viewModel = ServerPhotosViewModel()
    @State private var selectedPhoto: ServerPhoto?
    @State private var showRestoreConfirmation = false

    var body: some View {
        NavigationStack {
            ZStack {
                if viewModel.isLoading {
                    ProgressView("Loading server photos...")
                } else if viewModel.serverPhotos.isEmpty {
                    emptyStateView
                } else {
                    photoGridView
                }
            }
            .navigationTitle("Server-Only Photos")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        Task {
                            await viewModel.loadServerPhotos()
                        }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                    .disabled(viewModel.isLoading)
                }
            }
            .task {
                await viewModel.loadServerPhotos()
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

    private var photoGridView: some View {
        ScrollView {
            LazyVGrid(columns: [
                GridItem(.adaptive(minimum: 100), spacing: 4)
            ], spacing: 4) {
                ForEach(viewModel.serverPhotos) { photoWithThumbnail in
                    ServerPhotoCard(
                        photo: photoWithThumbnail.photo,
                        thumbnail: photoWithThumbnail.thumbnail,
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

struct ServerPhotoCard: View {
    let photo: ServerPhoto
    let thumbnail: UIImage?
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

                // Restore button overlay
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
            .frame(width: 100, height: 100)

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
        .shadow(radius: 2)
    }
}

#Preview {
    ServerPhotosView()
}
