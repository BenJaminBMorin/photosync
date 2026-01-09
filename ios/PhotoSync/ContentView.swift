import SwiftUI

struct ContentView: View {
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            // Photos on Device - Local photos only
            GalleryView()
                .tabItem {
                    Label("On Device", systemImage: "iphone")
                }
                .tag(0)

            // Photos in Cloud - Server-only photos
            ServerPhotosView()
                .tabItem {
                    Label("In Cloud", systemImage: "cloud.fill")
                }
                .tag(1)

            // All Pictures - Combined view (future enhancement)
            GalleryView()
                .tabItem {
                    Label("All Photos", systemImage: "photo.on.rectangle.angled")
                }
                .tag(2)

            // Settings
            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gear")
                }
                .tag(3)
        }
        .onAppear {
            // Log crash detection but don't show popup
            Task {
                if await Logger.shared.didCrashLastSession() {
                    await Logger.shared.info("Detected crash from previous session (crash report popup disabled)")
                }
            }
        }
    }
}

#Preview {
    ContentView()
        .environment(\.managedObjectContext, PersistenceController.preview.container.viewContext)
}
