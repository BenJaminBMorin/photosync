import SwiftUI

@main
struct PhotoSyncApp: App {
    let persistenceController = PersistenceController.shared

    init() {
        // Initialize logger early and log app launch
        Task {
            await Logger.shared.info("PhotoSync app launched")
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(\.managedObjectContext, persistenceController.container.viewContext)
                .onAppear {
                    Task {
                        await Logger.shared.info("Main view appeared")
                    }
                }
        }
    }
}
