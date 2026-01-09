import SwiftUI
import Photos

struct PhotoGridItem: View {
    let photoState: PhotoWithState
    let onTap: () -> Void
    let onIgnoreTap: () -> Void
    let onDeleteTap: () -> Void

    @State private var thumbnail: UIImage?
    @State private var showButtons = false

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
                                    colors: [Color.gray.opacity(0.2), Color.gray.opacity(0.1)],
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

                // Selection overlay with rounded corners
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

                    // Selection checkmark with better styling
                    HStack {
                        VStack {
                            ZStack {
                                Circle()
                                    .fill(.blue)
                                    .frame(width: 28, height: 28)
                                    .shadow(color: .black.opacity(0.2), radius: 2, x: 0, y: 1)

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

                // Animated sync badge - show for all non-ignored sync states
                if photoState.syncState != .ignored {
                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            AnimatedSyncBadge(state: photoState.badgeState)
                                .padding(4)
                        }
                    }
                    .frame(width: geometry.size.width, height: geometry.size.width)
                }

                // Ignored indicator
                if photoState.syncState == .ignored {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
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

                // Action buttons overlay (show on long press or when selected)
                if showButtons && !photoState.isSelected {
                    VStack {
                        Spacer()
                        HStack(spacing: 4) {
                            // Hide button
                            Button {
                                onIgnoreTap()
                                showButtons = false
                            } label: {
                                VStack(spacing: 2) {
                                    Image(systemName: photoState.syncState == .ignored ? "eye" : "eye.slash")
                                        .font(.system(size: 14))
                                    Text(photoState.syncState == .ignored ? "Show" : "Hide")
                                        .font(.system(size: 10))
                                }
                                .foregroundColor(.white)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 6)
                                .background(Color.blue.opacity(0.9))
                            }

                            // Delete button
                            Button {
                                onDeleteTap()
                                showButtons = false
                            } label: {
                                VStack(spacing: 2) {
                                    Image(systemName: "trash")
                                        .font(.system(size: 14))
                                    Text("Delete")
                                        .font(.system(size: 10))
                                }
                                .foregroundColor(.white)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 6)
                                .background(Color.red.opacity(0.9))
                            }
                        }
                    }
                    .frame(width: geometry.size.width, height: geometry.size.width)
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .onTapGesture {
            if showButtons {
                showButtons = false
            } else {
                onTap()
            }
        }
        .onLongPressGesture(minimumDuration: 0.3) {
            showButtons.toggle()
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
        let size = CGSize(width: 200, height: 200)
        thumbnail = await PhotoLibraryService.shared.getThumbnail(
            for: photoState.photo.asset,
            size: size
        )
    }
}

// MARK: - Animated Sync Badge Component

private struct AnimatedSyncBadge: View {
    let state: SyncBadgeState
    @State private var isAnimating = false

    var body: some View {
        Circle()
            .fill(badgeColor)
            .frame(width: 20, height: 20)
            .overlay {
                badgeIcon
                    .font(.system(size: iconSize).bold())
                    .foregroundColor(.white)
            }
            .scaleEffect(isAnimating && state == .syncing ? 1.1 : 1.0)
            .opacity(isAnimating && state == .syncing ? 0.8 : 1.0)
            .onAppear {
                if state == .syncing {
                    withAnimation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true)) {
                        isAnimating = true
                    }
                }
            }
            .onChange(of: state) { oldValue, newValue in
                if newValue == .syncing {
                    withAnimation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true)) {
                        isAnimating = true
                    }
                } else {
                    // Stop animating and do a completion animation
                    withAnimation(.easeOut(duration: 0.3)) {
                        isAnimating = false
                    }
                }
            }
    }

    private var badgeColor: Color {
        switch state {
        case .queued:
            return Color.orange
        case .syncing:
            return Color.blue
        case .synced:
            return Color.green
        case .error:
            return Color.red
        }
    }

    private var badgeIcon: some View {
        Group {
            switch state {
            case .queued:
                Image(systemName: "clock.fill")
            case .syncing:
                Image(systemName: "icloud.and.arrow.up")
            case .synced:
                Image(systemName: "checkmark")
            case .error:
                Image(systemName: "exclamationmark")
            }
        }
    }

    private var iconSize: CGFloat {
        switch state {
        case .queued:
            return 9
        case .syncing:
            return 10
        case .synced, .error:
            return 11
        }
    }
}

#Preview {
    PhotoGridItem(
        photoState: PhotoWithState(
            photo: Photo(asset: PHAsset(), isSynced: false),
            isSelected: true
        ),
        onTap: {},
        onIgnoreTap: {},
        onDeleteTap: {}
    )
    .frame(width: 120, height: 120)
}
