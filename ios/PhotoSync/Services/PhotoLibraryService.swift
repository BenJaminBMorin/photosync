import Foundation
import Photos
import UIKit

/// Service for accessing the device's photo library
actor PhotoLibraryService {
    static let shared = PhotoLibraryService()

    private init() {}

    /// Request photo library authorization
    func requestAuthorization() async -> PHAuthorizationStatus {
        await PHPhotoLibrary.requestAuthorization(for: .readWrite)
    }

    /// Check current authorization status
    var authorizationStatus: PHAuthorizationStatus {
        PHPhotoLibrary.authorizationStatus(for: .readWrite)
    }

    /// Fetch all photos from the library
    func fetchAllPhotos() async -> [PHAsset] {
        let fetchOptions = PHFetchOptions()
        fetchOptions.sortDescriptors = [NSSortDescriptor(key: "creationDate", ascending: false)]
        fetchOptions.predicate = NSPredicate(format: "mediaType == %d", PHAssetMediaType.image.rawValue)

        let fetchResult = PHAsset.fetchAssets(with: fetchOptions)

        var assets: [PHAsset] = []
        fetchResult.enumerateObjects { asset, _, _ in
            assets.append(asset)
        }

        return assets
    }

    /// Fetch photos from a specific collection
    func fetchPhotos(from collection: PHAssetCollection) async -> [PHAsset] {
        let fetchOptions = PHFetchOptions()
        fetchOptions.sortDescriptors = [NSSortDescriptor(key: "creationDate", ascending: false)]
        fetchOptions.predicate = NSPredicate(format: "mediaType == %d", PHAssetMediaType.image.rawValue)

        let fetchResult = PHAsset.fetchAssets(in: collection, options: fetchOptions)

        var assets: [PHAsset] = []
        fetchResult.enumerateObjects { asset, _, _ in
            assets.append(asset)
        }

        return assets
    }

    /// Fetch all collections (albums) from the library
    func fetchCollections() async -> [PhotoCollection] {
        var collections: [PhotoCollection] = []

        // User albums
        let userAlbums = PHAssetCollection.fetchAssetCollections(
            with: .album,
            subtype: .albumRegular,
            options: nil
        )
        userAlbums.enumerateObjects { collection, _, _ in
            let count = self.countPhotos(in: collection)
            if count > 0 {
                collections.append(PhotoCollection(collection: collection, photoCount: count))
            }
        }

        // Smart albums (Favorites, Screenshots, etc.)
        let smartAlbums = PHAssetCollection.fetchAssetCollections(
            with: .smartAlbum,
            subtype: .albumRegular,
            options: nil
        )
        smartAlbums.enumerateObjects { collection, _, _ in
            let count = self.countPhotos(in: collection)
            if count > 0 {
                collections.append(PhotoCollection(collection: collection, photoCount: count))
            }
        }

        return collections.sorted { $0.title < $1.title }
    }

    private func countPhotos(in collection: PHAssetCollection) -> Int {
        let fetchOptions = PHFetchOptions()
        fetchOptions.predicate = NSPredicate(format: "mediaType == %d", PHAssetMediaType.image.rawValue)
        let result = PHAsset.fetchAssets(in: collection, options: fetchOptions)
        return result.count
    }

    /// Get image data for a photo asset
    func getImageData(for asset: PHAsset) async throws -> Data {
        try await withCheckedThrowingContinuation { continuation in
            let options = PHImageRequestOptions()
            options.version = .current
            options.deliveryMode = .highQualityFormat
            options.isNetworkAccessAllowed = true
            options.isSynchronous = false

            PHImageManager.default().requestImageDataAndOrientation(
                for: asset,
                options: options
            ) { data, _, _, info in
                if let error = info?[PHImageErrorKey] as? Error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let imageData = data else {
                    continuation.resume(throwing: PhotoLibraryError.noImageData)
                    return
                }

                continuation.resume(returning: imageData)
            }
        }
    }

    /// Get thumbnail image for a photo asset
    func getThumbnail(for asset: PHAsset, size: CGSize) async -> UIImage? {
        await withCheckedContinuation { continuation in
            let options = PHImageRequestOptions()
            options.deliveryMode = .opportunistic
            options.isNetworkAccessAllowed = true
            options.isSynchronous = false

            PHImageManager.default().requestImage(
                for: asset,
                targetSize: size,
                contentMode: .aspectFill,
                options: options
            ) { image, info in
                // Only resume with the final image, not degraded ones
                let isDegraded = (info?[PHImageResultIsDegradedKey] as? Bool) ?? false
                if !isDegraded {
                    continuation.resume(returning: image)
                }
            }
        }
    }

    /// Get full resolution image for viewing
    func getFullResolutionImage(for asset: PHAsset, targetSize: CGSize) async -> UIImage? {
        await withCheckedContinuation { continuation in
            let options = PHImageRequestOptions()
            options.deliveryMode = .highQualityFormat
            options.isNetworkAccessAllowed = true
            options.isSynchronous = false
            options.resizeMode = .exact

            PHImageManager.default().requestImage(
                for: asset,
                targetSize: targetSize,
                contentMode: .aspectFit,
                options: options
            ) { image, info in
                let isDegraded = (info?[PHImageResultIsDegradedKey] as? Bool) ?? false
                if !isDegraded {
                    continuation.resume(returning: image)
                }
            }
        }
    }

    /// Get the original filename for an asset
    func getFilename(for asset: PHAsset) async -> String {
        let resources = PHAssetResource.assetResources(for: asset)
        if let resource = resources.first {
            return resource.originalFilename
        }
        return "IMG_\(asset.localIdentifier.prefix(8)).jpg"
    }

    /// Save an image to the photo library
    func saveImage(_ image: UIImage, filename: String) async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            PHPhotoLibrary.shared().performChanges({
                let request = PHAssetChangeRequest.creationRequestForAsset(from: image)
                // Note: Cannot set original filename through public API
                // The photo will be saved with a system-generated name
            }) { success, error in
                if let error = error {
                    continuation.resume(throwing: error)
                } else if success {
                    continuation.resume()
                } else {
                    continuation.resume(throwing: PhotoLibraryError.saveFailed)
                }
            }
        }
    }
}

enum PhotoLibraryError: Error, LocalizedError {
    case noImageData
    case unauthorized
    case saveFailed

    var errorDescription: String? {
        switch self {
        case .noImageData:
            return "Could not retrieve image data"
        case .unauthorized:
            return "Photo library access not authorized"
        case .saveFailed:
            return "Failed to save image to photo library"
        }
    }
}
