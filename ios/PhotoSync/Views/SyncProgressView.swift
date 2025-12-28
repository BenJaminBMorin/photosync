import SwiftUI

struct SyncProgressView: View {
    let progress: SyncProgress?
    let onCancel: () -> Void

    var body: some View {
        ZStack {
            Color.black.opacity(0.5)
                .ignoresSafeArea()

            VStack(spacing: 24) {
                Text("Syncing Photos...")
                    .font(.headline)

                if let progress = progress {
                    VStack(spacing: 16) {
                        ProgressView(value: progress.progressPercent)
                            .progressViewStyle(.linear)

                        Text("\(progress.completed) / \(progress.total)")
                            .font(.title2.monospacedDigit())

                        if let fileName = progress.currentFileName {
                            Text(fileName)
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        }

                        if progress.failed > 0 {
                            Text("\(progress.failed) failed")
                                .font(.caption)
                                .foregroundColor(.red)
                        }
                    }
                } else {
                    ProgressView()
                }

                Button("Cancel") {
                    onCancel()
                }
                .buttonStyle(.bordered)
            }
            .padding(32)
            .background(
                RoundedRectangle(cornerRadius: 16)
                    .fill(Color(.systemBackground))
            )
            .padding(32)
        }
    }
}

#Preview {
    SyncProgressView(
        progress: SyncProgress(
            total: 100,
            completed: 45,
            failed: 2,
            currentFileName: "IMG_1234.jpg"
        ),
        onCancel: {}
    )
}
