using PhotoSync.Core.Interfaces;

namespace PhotoSync.Infrastructure.Services;

/// <summary>
/// File system implementation of photo storage with Year/Month organization.
/// </summary>
public sealed class PhotoStorageService : IPhotoStorageService
{
    private readonly string _basePath;
    private readonly HashSet<string> _allowedExtensions;
    private readonly long _maxFileSizeBytes;

    private static readonly HashSet<string> DefaultAllowedExtensions = new(StringComparer.OrdinalIgnoreCase)
    {
        ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp", ".tiff", ".tif"
    };

    public PhotoStorageService(string basePath, IEnumerable<string>? allowedExtensions = null, long maxFileSizeMB = 50)
    {
        if (string.IsNullOrWhiteSpace(basePath))
            throw new ArgumentException("Base path cannot be empty.", nameof(basePath));

        _basePath = Path.GetFullPath(basePath);
        _allowedExtensions = allowedExtensions?.ToHashSet(StringComparer.OrdinalIgnoreCase)
            ?? DefaultAllowedExtensions;
        _maxFileSizeBytes = maxFileSizeMB * 1024 * 1024;

        // Ensure base directory exists
        Directory.CreateDirectory(_basePath);
    }

    public async Task<string> StoreAsync(
        Stream fileStream,
        string originalFilename,
        DateTime dateTaken,
        CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(fileStream);

        if (string.IsNullOrWhiteSpace(originalFilename))
            throw new ArgumentException("Filename cannot be empty.", nameof(originalFilename));

        // Validate file size
        if (fileStream.CanSeek && fileStream.Length > _maxFileSizeBytes)
            throw new InvalidOperationException($"File size exceeds maximum allowed size of {_maxFileSizeBytes / (1024 * 1024)}MB.");

        // Sanitize and validate filename
        var sanitizedFilename = SanitizeFilename(originalFilename);
        var extension = Path.GetExtension(sanitizedFilename);

        if (!_allowedExtensions.Contains(extension))
            throw new InvalidOperationException($"File extension '{extension}' is not allowed.");

        // Create Year/Month folder structure
        var year = dateTaken.Year.ToString("D4");
        var month = dateTaken.Month.ToString("D2");
        var relativeFolderPath = Path.Combine(year, month);
        var absoluteFolderPath = Path.Combine(_basePath, relativeFolderPath);

        Directory.CreateDirectory(absoluteFolderPath);

        // Generate unique filename to prevent collisions
        var uniqueFilename = GenerateUniqueFilename(sanitizedFilename, absoluteFolderPath);
        var relativeFilePath = Path.Combine(relativeFolderPath, uniqueFilename);
        var absoluteFilePath = Path.Combine(_basePath, relativeFilePath);

        // Validate the final path is still within base path (security check)
        var normalizedAbsolutePath = Path.GetFullPath(absoluteFilePath);
        if (!normalizedAbsolutePath.StartsWith(_basePath, StringComparison.OrdinalIgnoreCase))
            throw new InvalidOperationException("Invalid file path detected.");

        // Write file to disk
        await using var fileStreamOut = new FileStream(
            absoluteFilePath,
            FileMode.CreateNew,
            FileAccess.Write,
            FileShare.None,
            bufferSize: 81920, // 80KB buffer for performance
            useAsync: true);

        await fileStream.CopyToAsync(fileStreamOut, cancellationToken);

        // Return path with forward slashes for consistency
        return relativeFilePath.Replace('\\', '/');
    }

    public bool Delete(string storedPath)
    {
        if (string.IsNullOrWhiteSpace(storedPath))
            return false;

        var fullPath = GetFullPath(storedPath);

        if (!File.Exists(fullPath))
            return false;

        File.Delete(fullPath);
        return true;
    }

    public string GetFullPath(string storedPath)
    {
        if (string.IsNullOrWhiteSpace(storedPath))
            throw new ArgumentException("Stored path cannot be empty.", nameof(storedPath));

        // Normalize path separators
        var normalizedPath = storedPath.Replace('/', Path.DirectorySeparatorChar);
        var fullPath = Path.GetFullPath(Path.Combine(_basePath, normalizedPath));

        // Security: ensure path is within base path
        if (!fullPath.StartsWith(_basePath, StringComparison.OrdinalIgnoreCase))
            throw new InvalidOperationException("Invalid stored path - path traversal detected.");

        return fullPath;
    }

    public bool Exists(string storedPath)
    {
        try
        {
            var fullPath = GetFullPath(storedPath);
            return File.Exists(fullPath);
        }
        catch
        {
            return false;
        }
    }

    /// <summary>
    /// Sanitizes a filename to remove path components and invalid characters.
    /// </summary>
    private static string SanitizeFilename(string filename)
    {
        // Get just the filename, no path
        var name = Path.GetFileName(filename);

        if (string.IsNullOrWhiteSpace(name))
            throw new ArgumentException("Invalid filename.");

        // Replace invalid characters
        var invalidChars = Path.GetInvalidFileNameChars();
        foreach (var c in invalidChars)
        {
            name = name.Replace(c, '_');
        }

        // Limit filename length
        const int maxLength = 200;
        if (name.Length > maxLength)
        {
            var ext = Path.GetExtension(name);
            var nameWithoutExt = Path.GetFileNameWithoutExtension(name);
            name = nameWithoutExt[..(maxLength - ext.Length)] + ext;
        }

        return name;
    }

    /// <summary>
    /// Generates a unique filename by appending a suffix if necessary.
    /// </summary>
    private static string GenerateUniqueFilename(string filename, string folderPath)
    {
        var nameWithoutExt = Path.GetFileNameWithoutExtension(filename);
        var extension = Path.GetExtension(filename);
        var candidate = filename;
        var counter = 1;

        while (File.Exists(Path.Combine(folderPath, candidate)))
        {
            // Append unique suffix: originalname_001.jpg, originalname_002.jpg, etc.
            candidate = $"{nameWithoutExt}_{counter:D3}{extension}";
            counter++;

            // Safety limit
            if (counter > 9999)
            {
                // Fall back to GUID
                candidate = $"{nameWithoutExt}_{Guid.NewGuid():N}{extension}";
                break;
            }
        }

        return candidate;
    }
}
