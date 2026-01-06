import SwiftUI

struct ContentView: View {
    @State private var selectedTab = 0
    @State private var showCrashReport = false
    @State private var crashLogURL: URL?

    var body: some View {
        TabView(selection: $selectedTab) {
            GalleryView()
                .tabItem {
                    Label("Photos", systemImage: "photo.on.rectangle")
                }
                .tag(0)

            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gear")
                }
                .tag(1)

            LogsView()
                .tabItem {
                    Label("Logs", systemImage: "doc.text")
                }
                .tag(2)
        }
        .sheet(isPresented: $showCrashReport) {
            CrashReportView(logURL: crashLogURL)
        }
        .onAppear {
            checkForCrash()
        }
    }

    private func checkForCrash() {
        Task {
            if await Logger.shared.didCrashLastSession() {
                crashLogURL = await Logger.shared.getPreviousSessionLog()
                showCrashReport = true
                await Logger.shared.info("Detected crash from previous session, showing crash report")
            }
        }
    }
}

#Preview {
    ContentView()
        .environment(\.managedObjectContext, PersistenceController.preview.container.viewContext)
}
