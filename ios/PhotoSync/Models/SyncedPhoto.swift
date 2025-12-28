import Foundation
import CoreData

/// Core Data extension for SyncedPhotoEntity
extension SyncedPhotoEntity {
    /// Create a new synced photo record
    static func create(
        context: NSManagedObjectContext,
        localIdentifier: String,
        serverPhotoId: String,
        displayName: String,
        dateTaken: Date
    ) -> SyncedPhotoEntity {
        let entity = SyncedPhotoEntity(context: context)
        entity.localIdentifier = localIdentifier
        entity.serverPhotoId = serverPhotoId
        entity.displayName = displayName
        entity.dateTaken = dateTaken
        entity.syncedAt = Date()
        return entity
    }

    /// Check if a photo with the given local identifier has been synced
    static func isSynced(localIdentifier: String, context: NSManagedObjectContext) -> Bool {
        let request = SyncedPhotoEntity.fetchRequest()
        request.predicate = NSPredicate(format: "localIdentifier == %@", localIdentifier)
        request.fetchLimit = 1

        do {
            let count = try context.count(for: request)
            return count > 0
        } catch {
            print("Error checking sync status: \(error)")
            return false
        }
    }

    /// Get all synced local identifiers
    static func allSyncedIdentifiers(context: NSManagedObjectContext) -> Set<String> {
        let request = SyncedPhotoEntity.fetchRequest()
        request.propertiesToFetch = ["localIdentifier"]

        do {
            let results = try context.fetch(request)
            return Set(results.compactMap { $0.localIdentifier })
        } catch {
            print("Error fetching synced identifiers: \(error)")
            return []
        }
    }

    /// Get count of synced photos
    static func syncedCount(context: NSManagedObjectContext) -> Int {
        let request = SyncedPhotoEntity.fetchRequest()
        do {
            return try context.count(for: request)
        } catch {
            return 0
        }
    }
}
