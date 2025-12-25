namespace PhotoSync.Api.Endpoints;

/// <summary>
/// Health check endpoint definitions.
/// </summary>
public static class HealthEndpoints
{
    public static void MapHealthEndpoints(this IEndpointRouteBuilder routes)
    {
        routes.MapGet("/api/health", () => Results.Ok(new
        {
            status = "healthy",
            timestamp = DateTime.UtcNow
        }))
        .WithName("HealthCheck")
        .WithTags("Health")
        .AllowAnonymous();

        // Also map to /health for convenience
        routes.MapGet("/health", () => Results.Ok(new
        {
            status = "healthy",
            timestamp = DateTime.UtcNow
        }))
        .WithName("HealthCheckShort")
        .WithTags("Health")
        .AllowAnonymous();
    }
}
