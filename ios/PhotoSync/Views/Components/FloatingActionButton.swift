import SwiftUI

/// Floating action button that appears when photos are selected
struct FloatingActionButton: View {
    let selectedCount: Int
    let onSyncTap: () -> Void
    let onHideTap: () -> Void
    let onDeleteTap: () -> Void
    let onClearSelection: () -> Void

    @State private var isExpanded = false

    var body: some View {
        VStack {
            Spacer()

            HStack {
                Spacer()

                VStack(spacing: 12) {
                    // Action buttons (shown when expanded)
                    if isExpanded {
                        actionButtons
                            .transition(.asymmetric(
                                insertion: .scale.combined(with: .opacity),
                                removal: .scale.combined(with: .opacity)
                            ))
                    }

                    // Main FAB
                    Button {
                        withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                            isExpanded.toggle()
                        }
                        // Haptic feedback
                        let impact = UIImpactFeedbackGenerator(style: .medium)
                        impact.impactOccurred()
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
                                Image(systemName: isExpanded ? "xmark" : "checkmark.circle")
                                    .font(.system(size: 20, weight: .semibold))
                                    .rotationEffect(.degrees(isExpanded ? 90 : 0))

                                if !isExpanded {
                                    Text("\(selectedCount)")
                                        .font(.caption2.bold())
                                        .monospacedDigit()
                                }
                            }
                            .foregroundColor(.white)
                        }
                    }
                }
                .padding(.trailing, 20)
                .padding(.bottom, 20)
            }
        }
    }

    private var actionButtons: some View {
        VStack(spacing: 12) {
            // Sync button
            ActionButton(
                icon: "icloud.and.arrow.up",
                label: "Sync",
                color: .green
            ) {
                onSyncTap()
                withAnimation(.spring(response: 0.3)) {
                    isExpanded = false
                }
            }

            // Hide button
            ActionButton(
                icon: "eye.slash",
                label: "Hide",
                color: .orange
            ) {
                onHideTap()
                withAnimation(.spring(response: 0.3)) {
                    isExpanded = false
                }
            }

            // Delete button
            ActionButton(
                icon: "trash",
                label: "Delete",
                color: .red
            ) {
                onDeleteTap()
                withAnimation(.spring(response: 0.3)) {
                    isExpanded = false
                }
            }

            // Clear selection button
            ActionButton(
                icon: "xmark.circle",
                label: "Clear",
                color: .gray
            ) {
                onClearSelection()
                withAnimation(.spring(response: 0.3)) {
                    isExpanded = false
                }
            }
        }
    }
}

// MARK: - Action Button

private struct ActionButton: View {
    let icon: String
    let label: String
    let color: Color
    let action: () -> Void

    var body: some View {
        Button(action: {
            let impact = UIImpactFeedbackGenerator(style: .light)
            impact.impactOccurred()
            action()
        }) {
            HStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.system(size: 16, weight: .semibold))
                    .frame(width: 20)

                Text(label)
                    .font(.subheadline.weight(.semibold))
            }
            .foregroundColor(.white)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [color, color.opacity(0.8)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                    .shadow(color: color.opacity(0.4), radius: 4, x: 0, y: 2)
            )
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Preview

#Preview {
    ZStack {
        Color.gray.opacity(0.2)
            .ignoresSafeArea()

        FloatingActionButton(
            selectedCount: 12,
            onSyncTap: {},
            onHideTap: {},
            onDeleteTap: {},
            onClearSelection: {}
        )
    }
}
