namespace PhotoSync.Core.Entities;

/// <summary>
/// Represents a synced photo stored on the server.
/// </summary>
public sealed class Photo
{
    /// <summary>
    /// Unique identifier for the photo.
    /// </summary>
    public Guid Id { get; private set; }

    /// <summary>
    /// Original filename from the device.
    /// </summary>
    public string OriginalFilename { get; private set; } = string.Empty;

    /// <summary>
    /// Relative path where the file is stored (e.g., "2024/03/IMG_xxx.jpg").
    /// </summary>
    public string StoredPath { get; private set; } = string.Empty;

    /// <summary>
    /// SHA256 hash of the file content for duplicate detection.
    /// </summary>
    public string FileHash { get; private set; } = string.Empty;

    /// <summary>
    /// Size of the file in bytes.
    /// </summary>
    public long FileSize { get; private set; }

    /// <summary>
    /// Date the photo was taken (from device metadata).
    /// </summary>
    public DateTime DateTaken { get; private set; }

    /// <summary>
    /// Date/time when the photo was uploaded to the server.
    /// </summary>
    public DateTime UploadedAt { get; private set; }

    // Private constructor for EF Core
    private Photo() { }

    /// <summary>
    /// Creates a new Photo entity.
    /// </summary>
    public static Photo Create(
        string originalFilename,
        string storedPath,
        string fileHash,
        long fileSize,
        DateTime dateTaken)
    {
        if (string.IsNullOrWhiteSpace(originalFilename))
            throw new ArgumentException("Original filename cannot be empty.", nameof(originalFilename));

        if (string.IsNullOrWhiteSpace(storedPath))
            throw new ArgumentException("Stored path cannot be empty.", nameof(storedPath));

        if (string.IsNullOrWhiteSpace(fileHash))
            throw new ArgumentException("File hash cannot be empty.", nameof(fileHash));

        if (fileSize <= 0)
            throw new ArgumentException("File size must be positive.", nameof(fileSize));

        return new Photo
        {
            Id = Guid.NewGuid(),
            OriginalFilename = SanitizeFilename(originalFilename),
            StoredPath = storedPath,
            FileHash = fileHash.ToLowerInvariant(),
            FileSize = fileSize,
            DateTaken = dateTaken,
            UploadedAt = DateTime.UtcNow
        };
    }

    /// <summary>
    /// Sanitizes a filename to prevent path traversal attacks.
    /// </summary>
    private static string SanitizeFilename(string filename)
    {
        // Remove any path components - only keep the filename
        var name = Path.GetFileName(filename);

        // Remove any potentially dangerous characters
        var invalidChars = Path.GetInvalidFileNameChars();
        foreach (var c in invalidChars)
        {
            name = name.Replace(c, '_');
        }

        return name;
    }
}
