using System.ComponentModel.DataAnnotations;

namespace PhotoSync.Api.Configuration;

/// <summary>
/// Configuration options for photo storage.
/// </summary>
public sealed class PhotoStorageOptions
{
    public const string SectionName = "PhotoStorage";

    /// <summary>
    /// Base path where photos will be stored.
    /// </summary>
    [Required]
    public string BasePath { get; set; } = "./photos";

    /// <summary>
    /// Maximum allowed file size in megabytes.
    /// </summary>
    [Range(1, 500)]
    public int MaxFileSizeMB { get; set; } = 50;

    /// <summary>
    /// Allowed file extensions (including the dot).
    /// </summary>
    public string[] AllowedExtensions { get; set; } =
    [
        ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif"
    ];
}
