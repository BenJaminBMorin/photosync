namespace PhotoSync.Core.DTOs;

/// <summary>
/// Result of checking which file hashes already exist on the server.
/// </summary>
public sealed record CheckHashesResult
{
    /// <summary>
    /// Hashes that already exist on the server.
    /// </summary>
    public required IReadOnlyList<string> Existing { get; init; }

    /// <summary>
    /// Hashes that do not exist on the server (need to be uploaded).
    /// </summary>
    public required IReadOnlyList<string> Missing { get; init; }
}
