import SwiftUI
import Photos

struct CollectionsView: View {
    @StateObject private var viewModel = CollectionsViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.isLoading {
                    ProgressView("Loading collections...")
                } else if viewModel.collections.isEmpty {
                    ContentUnavailableView(
                        "No Collections",
                        systemImage: "photo.on.rectangle.angled",
                        description: Text("No photo albums found on your device")
                    )
                } else {
                    collectionsList
                }
            }
            .navigationTitle("Collections")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
            }
            .task {
                await viewModel.loadCollections()
            }
        }
    }

    private var collectionsList: some View {
        List {
            // All Photos option
            Button {
                viewModel.selectCollection(nil)
                dismiss()
            } label: {
                HStack {
                    Image(systemName: "photo.on.rectangle")
                        .font(.title2)
                        .foregroundColor(.blue)
                        .frame(width: 60, height: 60)
                        .background(Color.blue.opacity(0.1))
                        .cornerRadius(8)

                    VStack(alignment: .leading, spacing: 4) {
                        Text("All Photos")
                            .font(.headline)
                        Text("\(viewModel.totalPhotoCount) photos")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    Spacer()

                    if viewModel.selectedCollectionId == nil {
                        Image(systemName: "checkmark")
                            .foregroundColor(.blue)
                    }
                }
            }
            .buttonStyle(.plain)

            // Individual collections
            ForEach(viewModel.collections) { collection in
                Button {
                    viewModel.selectCollection(collection.id)
                    dismiss()
                } label: {
                    HStack {
                        CollectionThumbnailView(collection: collection.collection)
                            .frame(width: 60, height: 60)
                            .cornerRadius(8)

                        VStack(alignment: .leading, spacing: 4) {
                            Text(collection.title)
                                .font(.headline)
                            Text("\(collection.photoCount) photos")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        Spacer()

                        if viewModel.selectedCollectionId == collection.id {
                            Image(systemName: "checkmark")
                                .foregroundColor(.blue)
                        }
                    }
                }
                .buttonStyle(.plain)
            }
        }
    }
}

// Thumbnail view for collection
struct CollectionThumbnailView: View {
    let collection: PHAssetCollection
    @State private var thumbnail: UIImage?

    var body: some View {
        Group {
            if let thumbnail = thumbnail {
                Image(uiImage: thumbnail)
                    .resizable()
                    .aspectRatio(contentMode: .fill)
            } else {
                Rectangle()
                    .fill(Color.gray.opacity(0.2))
                    .overlay {
                        Image(systemName: "photo")
                            .foregroundColor(.gray)
                    }
            }
        }
        .task {
            await loadThumbnail()
        }
    }

    private func loadThumbnail() async {
        let fetchOptions = PHFetchOptions()
        fetchOptions.fetchLimit = 1
        fetchOptions.sortDescriptors = [NSSortDescriptor(key: "creationDate", ascending: false)]

        let result = PHAsset.fetchAssets(in: collection, options: fetchOptions)
        guard let asset = result.firstObject else { return }

        thumbnail = await PhotoLibraryService.shared.getThumbnail(
            for: asset,
            size: CGSize(width: 120, height: 120)
        )
    }
}

@MainActor
class CollectionsViewModel: ObservableObject {
    @Published var collections: [PhotoCollection] = []
    @Published var isLoading = false
    @Published var selectedCollectionId: String?
    @Published var totalPhotoCount: Int = 0

    private let photoLibrary = PhotoLibraryService.shared

    init() {
        selectedCollectionId = UserDefaults.standard.string(forKey: "selectedCollectionId")
    }

    func loadCollections() async {
        isLoading = true
        collections = await photoLibrary.fetchCollections()

        // Get total photo count
        let allPhotos = await photoLibrary.fetchAllPhotos()
        totalPhotoCount = allPhotos.count

        isLoading = false
    }

    func selectCollection(_ collectionId: String?) {
        selectedCollectionId = collectionId
        if let collectionId = collectionId {
            UserDefaults.standard.set(collectionId, forKey: "selectedCollectionId")
        } else {
            UserDefaults.standard.removeObject(forKey: "selectedCollectionId")
        }
        NotificationCenter.default.post(name: .collectionDidChange, object: nil)
    }
}

extension Notification.Name {
    static let collectionDidChange = Notification.Name("collectionDidChange")
}

#Preview {
    CollectionsView()
}
