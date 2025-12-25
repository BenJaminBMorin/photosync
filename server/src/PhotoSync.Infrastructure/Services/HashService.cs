using System.Security.Cryptography;
using System.Text.RegularExpressions;
using PhotoSync.Core.Interfaces;

namespace PhotoSync.Infrastructure.Services;

/// <summary>
/// SHA256-based hash service for file content hashing.
/// </summary>
public sealed partial class HashService : IHashService
{
    // Precompiled regex for SHA256 validation (64 hex characters)
    [GeneratedRegex("^[a-f0-9]{64}$", RegexOptions.IgnoreCase | RegexOptions.Compiled)]
    private static partial Regex Sha256Regex();

    public async Task<string> ComputeHashAsync(Stream stream, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(stream);

        if (!stream.CanRead)
            throw new ArgumentException("Stream must be readable.", nameof(stream));

        // Remember position if seekable
        var originalPosition = stream.CanSeek ? stream.Position : -1;

        try
        {
            var hashBytes = await SHA256.HashDataAsync(stream, cancellationToken);
            return Convert.ToHexString(hashBytes).ToLowerInvariant();
        }
        finally
        {
            // Reset position if the stream supports seeking
            if (stream.CanSeek && originalPosition >= 0)
            {
                stream.Position = originalPosition;
            }
        }
    }

    public string NormalizeHash(string hash)
    {
        if (string.IsNullOrWhiteSpace(hash))
            return string.Empty;

        // Remove any "sha256:" prefix if present
        var normalized = hash.Trim();
        if (normalized.StartsWith("sha256:", StringComparison.OrdinalIgnoreCase))
        {
            normalized = normalized[7..];
        }

        return normalized.ToLowerInvariant();
    }

    public bool IsValidHash(string hash)
    {
        if (string.IsNullOrWhiteSpace(hash))
            return false;

        var normalized = NormalizeHash(hash);
        return Sha256Regex().IsMatch(normalized);
    }
}
