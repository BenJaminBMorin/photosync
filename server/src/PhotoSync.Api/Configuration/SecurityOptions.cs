using System.ComponentModel.DataAnnotations;

namespace PhotoSync.Api.Configuration;

/// <summary>
/// Security configuration options.
/// </summary>
public sealed class SecurityOptions
{
    public const string SectionName = "Security";

    /// <summary>
    /// API key required for authentication.
    /// Must be at least 32 characters for security.
    /// </summary>
    [Required]
    [MinLength(32, ErrorMessage = "API key must be at least 32 characters for security.")]
    public string ApiKey { get; set; } = string.Empty;

    /// <summary>
    /// Header name for the API key.
    /// </summary>
    public string ApiKeyHeaderName { get; set; } = "X-API-Key";
}
