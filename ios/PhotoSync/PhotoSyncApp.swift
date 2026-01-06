import SwiftUI

@main
struct PhotoSyncApp: App {
    let persistenceController = PersistenceController.shared

    init() {
        // Initialize logger early
        _ = Logger.shared
        logInfo("PhotoSync app launched")
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(\.managedObjectContext, persistenceController.container.viewContext)
                .onAppear {
                    logInfo("Main view appeared")
                }
        }
    }
}
