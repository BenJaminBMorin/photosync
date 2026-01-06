import Foundation
import Photos

/// Represents a photo from the device's photo library
struct Photo: Identifiable, Hashable {
    let id: String
    let asset: PHAsset
    let creationDate: Date
    let isSynced: Bool

    var localIdentifier: String {
        asset.localIdentifier
    }

    init(asset: PHAsset, isSynced: Bool = false) {
        self.id = asset.localIdentifier
        self.asset = asset
        self.creationDate = asset.creationDate ?? Date()
        self.isSynced = isSynced
    }

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }

    static func == (lhs: Photo, rhs: Photo) -> Bool {
        lhs.id == rhs.id
    }
}

/// State of sync for a photo
enum SyncState: Equatable {
    case notSynced
    case syncing
    case synced
    case ignored
    case error(String)

    static func == (lhs: SyncState, rhs: SyncState) -> Bool {
        switch (lhs, rhs) {
        case (.notSynced, .notSynced), (.syncing, .syncing), (.synced, .synced), (.ignored, .ignored):
            return true
        case (.error(let lhsMsg), .error(let rhsMsg)):
            return lhsMsg == rhsMsg
        default:
            return false
        }
    }
}

/// Photo with its current selection and sync state for UI
struct PhotoWithState: Identifiable {
    let photo: Photo
    var isSelected: Bool
    var syncState: SyncState

    var id: String { photo.id }

    init(photo: Photo, isSelected: Bool = false, syncState: SyncState = .notSynced) {
        self.photo = photo
        self.isSelected = isSelected
        self.syncState = photo.isSynced ? .synced : .notSynced
    }
}

/// Group of photos by year and month
struct PhotoGroup: Identifiable {
    let id: String  // Format: "2024-01"
    let year: Int
    let month: Int
    let photos: [PhotoWithState]

    var displayTitle: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "MMMM yyyy"
        let date = Calendar.current.date(from: DateComponents(year: year, month: month)) ?? Date()
        return formatter.string(from: date)
    }

    var syncedCount: Int {
        photos.filter { $0.syncState == .synced }.count
    }

    var totalCount: Int {
        photos.count
    }
}
