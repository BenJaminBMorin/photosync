namespace PhotoSync.Core.Interfaces;

/// <summary>
/// Service for computing file hashes.
/// </summary>
public interface IHashService
{
    /// <summary>
    /// Computes the SHA256 hash of a stream.
    /// The stream position will be reset to the beginning after hashing.
    /// </summary>
    /// <param name="stream">The stream to hash.</param>
    /// <param name="cancellationToken">Cancellation token.</param>
    /// <returns>Lowercase hexadecimal hash string.</returns>
    Task<string> ComputeHashAsync(Stream stream, CancellationToken cancellationToken = default);

    /// <summary>
    /// Normalizes a hash string to lowercase for consistent comparison.
    /// </summary>
    string NormalizeHash(string hash);

    /// <summary>
    /// Validates that a string is a valid SHA256 hash format.
    /// </summary>
    bool IsValidHash(string hash);
}
