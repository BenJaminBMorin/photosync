import SwiftUI
import Photos

struct ImageViewerView: View {
    let photos: [PhotoWithState]
    @Binding var selectedIndex: Int
    @Environment(\.dismiss) var dismiss

    @State private var currentImage: UIImage?
    @State private var isLoading = true
    @State private var showShareSheet = false
    @State private var shareItems: [Any] = []
    @State private var showDeleteConfirmation = false

    private var currentPhoto: PhotoWithState? {
        guard selectedIndex >= 0 && selectedIndex < photos.count else { return nil }
        return photos[selectedIndex]
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Color.black.ignoresSafeArea()

                if let image = currentImage {
                    // Zoomable image view
                    TabView(selection: $selectedIndex) {
                        ForEach(Array(photos.enumerated()), id: \.element.id) { index, photo in
                            ZoomableImageView(image: image)
                                .tag(index)
                        }
                    }
                    .tabViewStyle(.page(indexDisplayMode: .automatic))
                    .onChange(of: selectedIndex) { _, _ in
                        loadCurrentImage()
                    }
                } else if isLoading {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                        .scaleEffect(1.5)
                } else {
                    VStack {
                        Image(systemName: "photo")
                            .font(.system(size: 64))
                            .foregroundColor(.gray)
                        Text("Unable to load image")
                            .foregroundColor(.gray)
                    }
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.title3)
                            .foregroundColor(.white)
                    }
                }

                ToolbarItem(placement: .principal) {
                    if let photo = currentPhoto {
                        VStack(spacing: 2) {
                            Text(formattedDate(photo.photo.creationDate))
                                .font(.subheadline.bold())
                                .foregroundColor(.white)
                            Text("\(selectedIndex + 1) of \(photos.count)")
                                .font(.caption)
                                .foregroundColor(.gray)
                        }
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button {
                            shareCurrentPhoto()
                        } label: {
                            Label("Share", systemImage: "square.and.arrow.up")
                        }

                        Button(role: .destructive) {
                            showDeleteConfirmation = true
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.title3)
                            .foregroundColor(.white)
                    }
                }
            }
            .toolbarBackground(.black, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
        }
        .task {
            await loadCurrentImage()
        }
        .sheet(isPresented: $showShareSheet) {
            if !shareItems.isEmpty {
                ShareSheet(items: shareItems)
            }
        }
        .alert("Delete Photo", isPresented: $showDeleteConfirmation) {
            Button("Cancel", role: .cancel) { }
            Button("Delete", role: .destructive) {
                deleteCurrentPhoto()
            }
        } message: {
            if let photo = currentPhoto {
                if photo.syncState == .synced {
                    Text("This photo is safely backed up on your server. It will be moved to Recently Deleted.")
                } else {
                    Text("WARNING: This photo has NOT been synced. If you delete it, it may be lost forever.")
                }
            }
        }
    }

    private func loadCurrentImage() {
        guard let photo = currentPhoto else { return }

        isLoading = true

        Task {
            let asset = photo.photo.asset
            let size = CGSize(width: UIScreen.main.bounds.width * 2, height: UIScreen.main.bounds.height * 2)

            let image = await PhotoLibraryService.shared.getFullResolutionImage(for: asset, targetSize: size)

            await MainActor.run {
                currentImage = image
                isLoading = false
            }
        }
    }

    private func shareCurrentPhoto() {
        guard let image = currentImage else { return }
        shareItems = [image]
        showShareSheet = true
    }

    private func deleteCurrentPhoto() {
        guard let photo = currentPhoto else { return }

        Task {
            do {
                try await PHPhotoLibrary.shared().performChanges {
                    PHAssetChangeRequest.deleteAssets([photo.photo.asset] as NSArray)
                }

                await MainActor.run {
                    // Move to next photo or dismiss if last
                    if photos.count <= 1 {
                        dismiss()
                    } else if selectedIndex >= photos.count - 1 {
                        selectedIndex = photos.count - 2
                    }
                }
            } catch {
                await Logger.shared.error("Failed to delete photo: \(error)")
            }
        }
    }

    private func formattedDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }
}

// MARK: - Zoomable Image View

struct ZoomableImageView: View {
    let image: UIImage

    @State private var scale: CGFloat = 1.0
    @State private var lastScale: CGFloat = 1.0
    @State private var offset: CGSize = .zero
    @State private var lastOffset: CGSize = .zero

    var body: some View {
        GeometryReader { geometry in
            Image(uiImage: image)
                .resizable()
                .aspectRatio(contentMode: .fit)
                .scaleEffect(scale)
                .offset(offset)
                .gesture(
                    MagnificationGesture()
                        .onChanged { value in
                            let delta = value / lastScale
                            lastScale = value
                            scale = min(max(scale * delta, 1), 4)
                        }
                        .onEnded { _ in
                            lastScale = 1.0
                            if scale < 1 {
                                withAnimation {
                                    scale = 1
                                    offset = .zero
                                }
                            }
                        }
                )
                .simultaneousGesture(
                    DragGesture()
                        .onChanged { value in
                            if scale > 1 {
                                offset = CGSize(
                                    width: lastOffset.width + value.translation.width,
                                    height: lastOffset.height + value.translation.height
                                )
                            }
                        }
                        .onEnded { _ in
                            lastOffset = offset
                        }
                )
                .onTapGesture(count: 2) {
                    withAnimation {
                        if scale > 1 {
                            scale = 1
                            offset = .zero
                            lastOffset = .zero
                        } else {
                            scale = 2
                        }
                    }
                }
                .frame(width: geometry.size.width, height: geometry.size.height)
        }
    }
}

// MARK: - Share Sheet

struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
