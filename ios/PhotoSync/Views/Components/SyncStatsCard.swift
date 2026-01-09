import SwiftUI

/// Hero stats card showing sync progress and quick filters
struct SyncStatsCard: View {
    let totalPhotos: Int
    let syncedCount: Int
    let unsyncedCount: Int
    let ignoredCount: Int
    @Binding var activeFilter: PhotoFilter
    @State private var isExpanded: Bool = true

    var syncProgress: Double {
        guard totalPhotos > 0 else { return 0 }
        return Double(syncedCount) / Double(totalPhotos)
    }

    var body: some View {
        VStack(spacing: 0) {
            if isExpanded {
                expandedView
                    .transition(.asymmetric(
                        insertion: .move(edge: .top).combined(with: .opacity),
                        removal: .move(edge: .top).combined(with: .opacity)
                    ))
            }

            // Collapse/expand button
            Button {
                withAnimation(.spring(response: 0.3)) {
                    isExpanded.toggle()
                }
            } label: {
                HStack {
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                    Text(isExpanded ? "Hide Stats" : "Show Stats")
                        .font(.caption)
                }
                .foregroundColor(.secondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .background(Color(.systemBackground))
            }
        }
        .background(Color(.systemGroupedBackground))
    }

    private var expandedView: some View {
        VStack(spacing: 16) {
            // Progress ring with stats
            HStack(spacing: 24) {
                // Circular progress
                ZStack {
                    Circle()
                        .stroke(Color.secondary.opacity(0.2), lineWidth: 8)
                        .frame(width: 80, height: 80)

                    Circle()
                        .trim(from: 0, to: syncProgress)
                        .stroke(
                            LinearGradient(
                                colors: [.blue, .green],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            style: StrokeStyle(lineWidth: 8, lineCap: .round)
                        )
                        .frame(width: 80, height: 80)
                        .rotationEffect(.degrees(-90))
                        .animation(.spring(response: 0.6), value: syncProgress)

                    VStack(spacing: 2) {
                        Text("\(Int(syncProgress * 100))%")
                            .font(.title3.bold())
                        Text("synced")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                // Stats breakdown
                VStack(alignment: .leading, spacing: 12) {
                    StatRow(
                        icon: "photo.stack",
                        color: .blue,
                        label: "Total",
                        value: "\(totalPhotos)"
                    )

                    StatRow(
                        icon: "checkmark.circle.fill",
                        color: .green,
                        label: "Synced",
                        value: "\(syncedCount)"
                    )

                    if unsyncedCount > 0 {
                        StatRow(
                            icon: "clock.fill",
                            color: .orange,
                            label: "Pending",
                            value: "\(unsyncedCount)"
                        )
                    }

                    if ignoredCount > 0 {
                        StatRow(
                            icon: "eye.slash.fill",
                            color: .gray,
                            label: "Hidden",
                            value: "\(ignoredCount)"
                        )
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(.horizontal)

            // Quick filter chips
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    FilterChip(
                        title: "All",
                        count: totalPhotos,
                        icon: "photo.on.rectangle",
                        isSelected: activeFilter == .all
                    ) {
                        activeFilter = .all
                    }

                    FilterChip(
                        title: "Unsynced",
                        count: unsyncedCount,
                        icon: "icloud.and.arrow.up",
                        isSelected: activeFilter == .unsynced
                    ) {
                        activeFilter = .unsynced
                    }

                    FilterChip(
                        title: "Today",
                        count: nil,
                        icon: "calendar",
                        isSelected: activeFilter == .today
                    ) {
                        activeFilter = .today
                    }

                    FilterChip(
                        title: "This Week",
                        count: nil,
                        icon: "calendar.badge.clock",
                        isSelected: activeFilter == .thisWeek
                    ) {
                        activeFilter = .thisWeek
                    }

                    if ignoredCount > 0 {
                        FilterChip(
                            title: "Hidden",
                            count: ignoredCount,
                            icon: "eye.slash",
                            isSelected: activeFilter == .hidden
                        ) {
                            activeFilter = .hidden
                        }
                    }
                }
                .padding(.horizontal)
            }
        }
        .padding(.vertical, 16)
    }
}

// MARK: - Supporting Views

private struct StatRow: View {
    let icon: String
    let color: Color
    let label: String
    let value: String

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: icon)
                .font(.caption)
                .foregroundColor(color)
                .frame(width: 16)

            Text(label)
                .font(.subheadline)
                .foregroundColor(.secondary)

            Spacer()

            Text(value)
                .font(.subheadline.bold())
                .monospacedDigit()
        }
    }
}

private struct FilterChip: View {
    let title: String
    let count: Int?
    let icon: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.caption)

                Text(title)
                    .font(.subheadline.weight(isSelected ? .semibold : .regular))

                if let count = count {
                    Text("\(count)")
                        .font(.caption)
                        .monospacedDigit()
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(
                            isSelected ? Color.white.opacity(0.3) : Color.secondary.opacity(0.2)
                        )
                        .clipShape(Capsule())
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(
                isSelected
                    ? LinearGradient(
                        colors: [.blue, .blue.opacity(0.8)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                    : LinearGradient(
                        colors: [.secondary.opacity(0.15), .secondary.opacity(0.1)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
            )
            .foregroundColor(isSelected ? .white : .primary)
            .clipShape(Capsule())
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Filter Enum

enum PhotoFilter {
    case all
    case unsynced
    case today
    case thisWeek
    case hidden
}

// MARK: - Preview

#Preview {
    SyncStatsCard(
        totalPhotos: 1250,
        syncedCount: 890,
        unsyncedCount: 360,
        ignoredCount: 25,
        activeFilter: .constant(.all)
    )
    .padding()
}
