import SwiftUI
import Photos

struct PhotoGridItem: View {
    let photoState: PhotoWithState
    let onTap: () -> Void
    let onIgnoreTap: () -> Void

    @State private var thumbnail: UIImage?

    var body: some View {
        GeometryReader { geometry in
            ZStack(alignment: .topLeading) {
                // Photo thumbnail
                if let thumbnail = thumbnail {
                    Image(uiImage: thumbnail)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(width: geometry.size.width, height: geometry.size.width)
                        .clipped()
                } else {
                    Rectangle()
                        .fill(Color.gray.opacity(0.2))
                        .frame(width: geometry.size.width, height: geometry.size.width)
                        .overlay {
                            ProgressView()
                        }
                }

                // Selection overlay
                if photoState.isSelected {
                    Rectangle()
                        .stroke(Color.accentColor, lineWidth: 3)
                        .frame(width: geometry.size.width, height: geometry.size.width)

                    // Selection checkmark
                    Circle()
                        .fill(Color.accentColor)
                        .frame(width: 24, height: 24)
                        .overlay {
                            Image(systemName: "checkmark")
                                .font(.caption.bold())
                                .foregroundColor(.white)
                        }
                        .padding(4)
                }

                // Synced indicator
                if photoState.syncState == .synced {
                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            Circle()
                                .fill(Color.green)
                                .frame(width: 20, height: 20)
                                .overlay {
                                    Image(systemName: "checkmark")
                                        .font(.caption2.bold())
                                        .foregroundColor(.white)
                                }
                                .padding(4)
                        }
                    }
                    .frame(width: geometry.size.width, height: geometry.size.width)
                }

                // Ignored indicator
                if photoState.syncState == .ignored {
                    Rectangle()
                        .fill(Color.black.opacity(0.6))
                        .frame(width: geometry.size.width, height: geometry.size.width)

                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            Circle()
                                .fill(Color.gray)
                                .frame(width: 20, height: 20)
                                .overlay {
                                    Image(systemName: "eye.slash")
                                        .font(.caption2.bold())
                                        .foregroundColor(.white)
                                }
                                .padding(4)
                        }
                    }
                    .frame(width: geometry.size.width, height: geometry.size.width)
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .contentShape(Rectangle())
        .onTapGesture {
            onTap()
        }
        .contextMenu {
            Button {
                onIgnoreTap()
            } label: {
                Label(
                    photoState.syncState == .ignored ? "Unignore" : "Ignore",
                    systemImage: photoState.syncState == .ignored ? "eye" : "eye.slash"
                )
            }
        }
        .task {
            await loadThumbnail()
        }
    }

    private func loadThumbnail() async {
        let size = CGSize(width: 200, height: 200)
        thumbnail = await PhotoLibraryService.shared.getThumbnail(
            for: photoState.photo.asset,
            size: size
        )
    }
}

#Preview {
    PhotoGridItem(
        photoState: PhotoWithState(
            photo: Photo(asset: PHAsset(), isSynced: false),
            isSelected: true
        ),
        onTap: {},
        onIgnoreTap: {}
    )
    .frame(width: 120, height: 120)
}
