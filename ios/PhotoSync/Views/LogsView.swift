import SwiftUI

struct LogsView: View {
    @State private var logs: [LogFile] = []
    @State private var selectedLog: LogFile?
    @State private var logContent: String = ""
    @State private var isLoading = false
    @State private var showShareSheet = false

    struct LogFile: Identifiable {
        let id = UUID()
        let url: URL
        let name: String
        let date: Date
        let size: String
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                if logs.isEmpty {
                    ContentUnavailableView(
                        "No Logs Available",
                        systemImage: "doc.text",
                        description: Text("Logs will appear here as the app runs")
                    )
                } else {
                    List {
                        Section("Current Session") {
                            if let currentLog = logs.first {
                                LogFileRow(logFile: currentLog, isCurrent: true)
                                    .onTapGesture {
                                        loadLog(currentLog)
                                    }
                            }
                        }

                        if logs.count > 1 {
                            Section("Previous Sessions") {
                                ForEach(Array(logs.dropFirst())) { logFile in
                                    LogFileRow(logFile: logFile, isCurrent: false)
                                        .onTapGesture {
                                            loadLog(logFile)
                                        }
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("Debug Logs")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        refreshLogs()
                    } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }

                if selectedLog != nil {
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button {
                            showShareSheet = true
                        } label: {
                            Image(systemName: "square.and.arrow.up")
                        }
                    }
                }
            }
            .sheet(item: $selectedLog) { logFile in
                LogDetailView(logFile: logFile, content: logContent)
            }
            .sheet(isPresented: $showShareSheet) {
                if let selectedLog = selectedLog {
                    ShareSheet(items: [selectedLog.url])
                }
            }
            .task {
                refreshLogs()
            }
        }
    }

    private func refreshLogs() {
        isLoading = true
        logs = Logger.shared.getAllLogURLs().map { url in
            let attributes = try? FileManager.default.attributesOfItem(atPath: url.path)
            let fileSize = attributes?[.size] as? Int64 ?? 0
            let sizeString = ByteCountFormatter.string(fromByteCount: fileSize, countStyle: .file)

            let date = (try? url.resourceValues(forKeys: [.creationDateKey]))?.creationDate ?? Date()

            return LogFile(
                url: url,
                name: url.lastPathComponent,
                date: date,
                size: sizeString
            )
        }
        isLoading = false
    }

    private func loadLog(_ logFile: LogFile) {
        logContent = Logger.shared.getLogContent(from: logFile.url) ?? "Failed to load log"
        selectedLog = logFile
    }
}

struct LogFileRow: View {
    let logFile: LogsView.LogFile
    let isCurrent: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(isCurrent ? "Current Session" : logFile.name)
                    .font(.headline)

                if isCurrent {
                    Image(systemName: "circle.fill")
                        .foregroundColor(.green)
                        .font(.system(size: 8))
                }

                Spacer()

                Text(logFile.size)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Text(logFile.date, style: .relative)
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}

struct LogDetailView: View {
    let logFile: LogsView.LogFile
    let content: String
    @Environment(\.dismiss) private var dismiss
    @State private var searchText = ""

    var filteredContent: String {
        if searchText.isEmpty {
            return content
        }
        return content.split(separator: "\n")
            .filter { $0.localizedCaseInsensitiveContains(searchText) }
            .joined(separator: "\n")
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Search bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search logs...", text: $searchText)
                        .textFieldStyle(.plain)

                    if !searchText.isEmpty {
                        Button {
                            searchText = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding()
                .background(Color(.systemGray6))

                // Log content
                ScrollView {
                    Text(filteredContent)
                        .font(.system(.caption, design: .monospaced))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                        .textSelection(.enabled)
                }
            }
            .navigationTitle(logFile.name)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Done") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    ShareLink(item: logFile.url) {
                        Image(systemName: "square.and.arrow.up")
                    }
                }
            }
        }
    }
}

struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

#Preview {
    LogsView()
}
