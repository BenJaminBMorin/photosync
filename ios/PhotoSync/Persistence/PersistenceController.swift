import CoreData

/// Core Data persistence controller
struct PersistenceController {
    static let shared = PersistenceController()

    let container: NSPersistentContainer

    /// Preview instance for SwiftUI previews
    static var preview: PersistenceController = {
        let controller = PersistenceController(inMemory: true)
        let viewContext = controller.container.viewContext

        // Create sample data for previews
        for i in 0..<5 {
            let photo = SyncedPhotoEntity(context: viewContext)
            photo.localIdentifier = "preview-\(i)"
            photo.serverPhotoId = UUID().uuidString
            photo.displayName = "Photo_\(i).jpg"
            photo.dateTaken = Date().addingTimeInterval(TimeInterval(-i * 86400))
            photo.syncedAt = Date()
        }

        do {
            try viewContext.save()
        } catch {
            fatalError("Failed to save preview context: \(error)")
        }

        return controller
    }()

    init(inMemory: Bool = false) {
        container = NSPersistentContainer(name: "PhotoSync")

        if inMemory {
            container.persistentStoreDescriptions.first?.url = URL(fileURLWithPath: "/dev/null")
        }

        container.loadPersistentStores { _, error in
            if let error = error as NSError? {
                fatalError("Failed to load Core Data store: \(error)")
            }
        }

        container.viewContext.automaticallyMergesChangesFromParent = true
        container.viewContext.mergePolicy = NSMergeByPropertyObjectTrumpMergePolicy
    }

    /// Save the view context if there are changes
    func save() {
        let context = container.viewContext
        if context.hasChanges {
            do {
                try context.save()
            } catch {
                print("Failed to save context: \(error)")
            }
        }
    }
}
