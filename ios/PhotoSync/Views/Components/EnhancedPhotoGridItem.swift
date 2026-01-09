import SwiftUI
import Photos

/// Enhanced photo grid item with better visuals and interactions
struct EnhancedPhotoGridItem: View {
    let photoState: PhotoWithState
    let onTap: () -> Void
    let onIgnoreTap: () -> Void
    let onDeleteTap: () -> Void

    @State private var thumbnail: UIImage?
    @State private var isPressed = false

    var body: some View {
        GeometryReader { geometry in
            ZStack(alignment: .topLeading) {
                // Photo thumbnail with rounded corners
                Group {
                    if let thumbnail = thumbnail {
                        Image(uiImage: thumbnail)
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geometry.size.width, height: geometry.size.width)
                            .clipped()
                    } else {
                        Rectangle()
                            .fill(
                                LinearGradient(
                                    colors: [
                                        Color.gray.opacity(0.2),
                                        Color.gray.opacity(0.1)
                                    ],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                )
                            )
                            .frame(width: geometry.size.width, height: geometry.size.width)
                            .overlay {
                                ProgressView()
                                    .tint(.secondary)
                            }
                    }
                }
                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))

                // Selection overlay with smooth animation
                if photoState.isSelected {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .strokeBorder(
                            LinearGradient(
                                colors: [.blue, .blue.opacity(0.8)],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            lineWidth: 3
                        )
                        .frame(width: geometry.size.width, height: geometry.size.width)
                        .animation(.spring(response: 0.3), value: photoState.isSelected)

                    // Checkmark badge
                    HStack {
                        VStack {
                            ZStack {
                                Circle()
                                    .fill(.blue)
                                    .frame(width: 28, height: 28)
                                    .shadow(color: .black.opacity(0.3), radius: 3, x: 0, y: 2)

                                Image(systemName: "checkmark")
                                    .font(.system(size: 14, weight: .bold))
                                    .foregroundColor(.white)
                            }
                            .padding(6)

                            Spacer()
                        }

                        Spacer()
                    }
                }

                // Sync state badge (always visible, top-right)
                if photoState.syncState != .ignored {
                    VStack {
                        HStack {
                            Spacer()

                            AnimatedSyncBadge(state: photoState.badgeState)
                                .shadow(color: .black.opacity(0.2), radius: 2, x: 0, y: 1)
                                .padding(6)
                        }

                        Spacer()
                    }
                }

                // Ignored overlay
                if photoState.syncState == .ignored {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(Color.black.opacity(0.6))
                        .frame(width: geometry.size.width, height: geometry.size.width)

                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            ZStack {
                                Circle()
                                    .fill(Color.gray)
                                    .frame(width: 24, height: 24)

                                Image(systemName: "eye.slash")
                                    .font(.system(size: 12, weight: .bold))
                                    .foregroundColor(.white)
                            }
                            .padding(6)
                        }
                    }
                }

                // Press effect
                if isPressed {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(Color.black.opacity(0.1))
                        .frame(width: geometry.size.width, height: geometry.size.width)
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .onTapGesture {
            // Haptic feedback
            let impact = UIImpactFeedbackGenerator(style: .light)
            impact.impactOccurred()
            onTap()
        }
        .simultaneousGesture(
            DragGesture(minimumDistance: 0)
                .onChanged { _ in
                    if !isPressed {
                        isPressed = true
                    }
                }
                .onEnded { _ in
                    isPressed = false
                }
        )
        .contextMenu {
            Button {
                onIgnoreTap()
            } label: {
                Label(
                    photoState.syncState == .ignored ? "Unignore" : "Ignore",
                    systemImage: photoState.syncState == .ignored ? "eye" : "eye.slash"
                )
            }

            Button(role: .destructive) {
                onDeleteTap()
            } label: {
                Label("Delete", systemImage: "trash")
            }
        }
        .task {
            await loadThumbnail()
        }
    }

    private func loadThumbnail() async {
        let size = CGSize(width: 300, height: 300)
        thumbnail = await PhotoLibraryService.shared.getThumbnail(
            for: photoState.photo.asset,
            size: size
        )
    }
}

#Preview {
    HStack(spacing: 4) {
        EnhancedPhotoGridItem(
            photoState: PhotoWithState(
                photo: Photo(asset: PHAsset(), isSynced: false),
                isSelected: false
            ),
            onTap: {},
            onIgnoreTap: {},
            onDeleteTap: {}
        )

        EnhancedPhotoGridItem(
            photoState: PhotoWithState(
                photo: Photo(asset: PHAsset(), isSynced: true),
                isSelected: true
            ),
            onTap: {},
            onIgnoreTap: {},
            onDeleteTap: {}
        )
    }
    .frame(height: 120)
    .padding()
}
