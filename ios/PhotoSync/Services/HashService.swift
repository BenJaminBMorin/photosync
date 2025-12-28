import Foundation
import CryptoKit

/// Service for computing SHA256 hashes of photo data
enum HashService {
    /// Compute SHA256 hash of data and return as lowercase hex string
    static func sha256(_ data: Data) -> String {
        let hash = SHA256.hash(data: data)
        return hash.compactMap { String(format: "%02x", $0) }.joined()
    }
}
