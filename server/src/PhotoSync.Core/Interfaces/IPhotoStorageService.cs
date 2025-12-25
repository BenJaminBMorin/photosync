namespace PhotoSync.Core.Interfaces;

/// <summary>
/// Service for storing and retrieving photo files.
/// </summary>
public interface IPhotoStorageService
{
    /// <summary>
    /// Stores a photo file and returns the relative storage path.
    /// The file will be organized into Year/Month folders based on dateTaken.
    /// </summary>
    /// <param name="fileStream">The file content stream.</param>
    /// <param name="originalFilename">Original filename from the device.</param>
    /// <param name="dateTaken">Date the photo was taken (for folder organization).</param>
    /// <param name="cancellationToken">Cancellation token.</param>
    /// <returns>Relative path where the file was stored (e.g., "2024/03/IMG_xxx.jpg").</returns>
    Task<string> StoreAsync(
        Stream fileStream,
        string originalFilename,
        DateTime dateTaken,
        CancellationToken cancellationToken = default);

    /// <summary>
    /// Deletes a photo file by its stored path.
    /// </summary>
    /// <param name="storedPath">Relative path of the file.</param>
    /// <returns>True if deleted, false if file didn't exist.</returns>
    bool Delete(string storedPath);

    /// <summary>
    /// Gets the full filesystem path for a stored photo.
    /// </summary>
    string GetFullPath(string storedPath);

    /// <summary>
    /// Checks if a file exists at the given stored path.
    /// </summary>
    bool Exists(string storedPath);
}
