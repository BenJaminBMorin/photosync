namespace PhotoSync.Core.DTOs;

/// <summary>
/// Result of a photo upload operation.
/// </summary>
public sealed record UploadResult
{
    /// <summary>
    /// Unique identifier of the photo.
    /// </summary>
    public required Guid Id { get; init; }

    /// <summary>
    /// Relative path where the photo is stored.
    /// </summary>
    public required string StoredPath { get; init; }

    /// <summary>
    /// When the photo was uploaded/recorded.
    /// </summary>
    public required DateTime UploadedAt { get; init; }

    /// <summary>
    /// True if this photo already existed (duplicate by hash).
    /// </summary>
    public required bool IsDuplicate { get; init; }

    /// <summary>
    /// Creates a result for a newly uploaded photo.
    /// </summary>
    public static UploadResult NewUpload(Guid id, string storedPath, DateTime uploadedAt)
        => new() { Id = id, StoredPath = storedPath, UploadedAt = uploadedAt, IsDuplicate = false };

    /// <summary>
    /// Creates a result for a duplicate photo.
    /// </summary>
    public static UploadResult Duplicate(Guid id, string storedPath, DateTime uploadedAt)
        => new() { Id = id, StoredPath = storedPath, UploadedAt = uploadedAt, IsDuplicate = true };
}
