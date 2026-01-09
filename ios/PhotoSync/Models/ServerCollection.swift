import Foundation

/// Represents a collection (album) on the server
struct ServerCollection: Identifiable, Codable, Hashable {
    let id: String
    let name: String
    let photoCount: Int
    let createdAt: Date
    let updatedAt: Date

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }

    static func == (lhs: ServerCollection, rhs: ServerCollection) -> Bool {
        lhs.id == rhs.id
    }
}

/// Request to create a new collection
struct CreateCollectionRequest: Codable {
    let name: String
}

/// Request to add/remove photos to/from a collection
struct ManageCollectionPhotosRequest: Codable {
    let photoIds: [String]
}

/// Response for collection operations
struct CollectionResponse: Codable {
    let collection: ServerCollection
}

/// Response for listing collections
struct CollectionsListResponse: Codable {
    let collections: [ServerCollection]
}
