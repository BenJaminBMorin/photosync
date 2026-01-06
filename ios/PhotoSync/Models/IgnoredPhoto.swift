import Foundation
import CoreData

/// Core Data extension for IgnoredPhotoEntity
extension IgnoredPhotoEntity {
    /// Create a new ignored photo record
    static func create(
        context: NSManagedObjectContext,
        localIdentifier: String
    ) -> IgnoredPhotoEntity {
        let entity = IgnoredPhotoEntity(context: context)
        entity.localIdentifier = localIdentifier
        entity.ignoredAt = Date()
        return entity
    }

    /// Check if a photo with the given local identifier is ignored
    static func isIgnored(localIdentifier: String, context: NSManagedObjectContext) -> Bool {
        let request = IgnoredPhotoEntity.fetchRequest()
        request.predicate = NSPredicate(format: "localIdentifier == %@", localIdentifier)
        request.fetchLimit = 1

        do {
            let count = try context.count(for: request)
            return count > 0
        } catch {
            print("Error checking ignored status: \(error)")
            return false
        }
    }

    /// Get all ignored local identifiers
    static func allIgnoredIdentifiers(context: NSManagedObjectContext) -> Set<String> {
        let request = IgnoredPhotoEntity.fetchRequest()
        request.propertiesToFetch = ["localIdentifier"]

        do {
            let results = try context.fetch(request)
            return Set(results.compactMap { $0.localIdentifier })
        } catch {
            print("Error fetching ignored identifiers: \(error)")
            return []
        }
    }

    /// Get count of ignored photos
    static func ignoredCount(context: NSManagedObjectContext) -> Int {
        let request = IgnoredPhotoEntity.fetchRequest()
        do {
            return try context.count(for: request)
        } catch {
            return 0
        }
    }

    /// Delete an ignored photo record
    static func unignore(localIdentifier: String, context: NSManagedObjectContext) {
        let request = IgnoredPhotoEntity.fetchRequest()
        request.predicate = NSPredicate(format: "localIdentifier == %@", localIdentifier)

        do {
            let results = try context.fetch(request)
            for entity in results {
                context.delete(entity)
            }
            try context.save()
        } catch {
            print("Error unignoring photo: \(error)")
        }
    }
}
