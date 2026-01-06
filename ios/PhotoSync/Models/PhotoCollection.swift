import Foundation
import Photos

/// Represents a photo collection (album) from the device
struct PhotoCollection: Identifiable, Hashable {
    let id: String
    let collection: PHAssetCollection
    let title: String
    let photoCount: Int

    init(collection: PHAssetCollection, photoCount: Int) {
        self.id = collection.localIdentifier
        self.collection = collection
        self.title = collection.localizedTitle ?? "Unknown Album"
        self.photoCount = photoCount
    }

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }

    static func == (lhs: PhotoCollection, rhs: PhotoCollection) -> Bool {
        lhs.id == rhs.id
    }
}
