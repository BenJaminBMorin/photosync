using System.Security.Cryptography;
using System.Text;
using Microsoft.Extensions.Options;
using PhotoSync.Api.Configuration;

namespace PhotoSync.Api.Middleware;

/// <summary>
/// Middleware for API key authentication.
/// </summary>
public sealed class ApiKeyAuthenticationMiddleware
{
    private readonly RequestDelegate _next;
    private readonly SecurityOptions _options;
    private readonly ILogger<ApiKeyAuthenticationMiddleware> _logger;

    // Endpoints that don't require authentication
    private static readonly HashSet<string> PublicEndpoints = new(StringComparer.OrdinalIgnoreCase)
    {
        "/api/health",
        "/health"
    };

    public ApiKeyAuthenticationMiddleware(
        RequestDelegate next,
        IOptions<SecurityOptions> options,
        ILogger<ApiKeyAuthenticationMiddleware> logger)
    {
        _next = next;
        _options = options.Value;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        var path = context.Request.Path.Value ?? string.Empty;

        // Skip authentication for public endpoints
        if (IsPublicEndpoint(path))
        {
            await _next(context);
            return;
        }

        // Only authenticate API routes
        if (!path.StartsWith("/api", StringComparison.OrdinalIgnoreCase))
        {
            await _next(context);
            return;
        }

        // Check for API key header
        if (!context.Request.Headers.TryGetValue(_options.ApiKeyHeaderName, out var providedApiKey))
        {
            _logger.LogWarning("API request without API key from {RemoteIp}",
                context.Connection.RemoteIpAddress);

            context.Response.StatusCode = StatusCodes.Status401Unauthorized;
            await context.Response.WriteAsJsonAsync(new { error = "API key is required." });
            return;
        }

        // Constant-time comparison to prevent timing attacks
        if (!ConstantTimeEquals(_options.ApiKey, providedApiKey.ToString()))
        {
            _logger.LogWarning("Invalid API key attempt from {RemoteIp}",
                context.Connection.RemoteIpAddress);

            context.Response.StatusCode = StatusCodes.Status401Unauthorized;
            await context.Response.WriteAsJsonAsync(new { error = "Invalid API key." });
            return;
        }

        await _next(context);
    }

    private static bool IsPublicEndpoint(string path)
    {
        return PublicEndpoints.Any(ep => path.Equals(ep, StringComparison.OrdinalIgnoreCase));
    }

    /// <summary>
    /// Constant-time string comparison to prevent timing attacks.
    /// </summary>
    private static bool ConstantTimeEquals(string a, string b)
    {
        var aBytes = Encoding.UTF8.GetBytes(a);
        var bBytes = Encoding.UTF8.GetBytes(b);

        return CryptographicOperations.FixedTimeEquals(aBytes, bBytes);
    }
}

/// <summary>
/// Extension methods for API key authentication.
/// </summary>
public static class ApiKeyAuthenticationExtensions
{
    public static IApplicationBuilder UseApiKeyAuthentication(this IApplicationBuilder app)
    {
        return app.UseMiddleware<ApiKeyAuthenticationMiddleware>();
    }
}
