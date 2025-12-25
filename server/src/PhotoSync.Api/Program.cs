using Microsoft.EntityFrameworkCore;
using PhotoSync.Api.Configuration;
using PhotoSync.Api.Endpoints;
using PhotoSync.Api.Middleware;
using PhotoSync.Core.Interfaces;
using PhotoSync.Infrastructure.Data;
using PhotoSync.Infrastructure.Services;

var builder = WebApplication.CreateBuilder(args);

// Configure options
builder.Services.AddOptions<PhotoStorageOptions>()
    .Bind(builder.Configuration.GetSection(PhotoStorageOptions.SectionName))
    .ValidateDataAnnotations()
    .ValidateOnStart();

builder.Services.AddOptions<SecurityOptions>()
    .Bind(builder.Configuration.GetSection(SecurityOptions.SectionName))
    .ValidateDataAnnotations()
    .ValidateOnStart();

// Configure database
var connectionString = builder.Configuration.GetConnectionString("PhotoDb")
    ?? "Data Source=photosync.db";

builder.Services.AddDbContext<PhotoDbContext>(options =>
    options.UseSqlite(connectionString));

// Register services
builder.Services.AddScoped<IPhotoRepository, PhotoRepository>();
builder.Services.AddSingleton<IHashService, HashService>();

// Register PhotoStorageService with configuration
builder.Services.AddSingleton<IPhotoStorageService>(sp =>
{
    var options = builder.Configuration
        .GetSection(PhotoStorageOptions.SectionName)
        .Get<PhotoStorageOptions>() ?? new PhotoStorageOptions();

    return new PhotoStorageService(
        options.BasePath,
        options.AllowedExtensions,
        options.MaxFileSizeMB);
});

// Configure request size limits for file uploads
builder.Services.Configure<Microsoft.AspNetCore.Http.Features.FormOptions>(options =>
{
    var storageOptions = builder.Configuration
        .GetSection(PhotoStorageOptions.SectionName)
        .Get<PhotoStorageOptions>() ?? new PhotoStorageOptions();

    options.MultipartBodyLengthLimit = storageOptions.MaxFileSizeMB * 1024L * 1024L;
});

builder.WebHost.ConfigureKestrel(options =>
{
    var storageOptions = builder.Configuration
        .GetSection(PhotoStorageOptions.SectionName)
        .Get<PhotoStorageOptions>() ?? new PhotoStorageOptions();

    options.Limits.MaxRequestBodySize = storageOptions.MaxFileSizeMB * 1024L * 1024L;
});

// Add logging
builder.Logging.AddConsole();

var app = builder.Build();

// Ensure database is created
using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<PhotoDbContext>();
    await db.Database.EnsureCreatedAsync();
}

// Configure middleware pipeline
app.UseApiKeyAuthentication();

// Map endpoints
app.MapHealthEndpoints();
app.MapPhotoEndpoints();

// Log startup
var logger = app.Services.GetRequiredService<ILogger<Program>>();
var storageConfig = builder.Configuration
    .GetSection(PhotoStorageOptions.SectionName)
    .Get<PhotoStorageOptions>();

logger.LogInformation("PhotoSync Server starting...");
logger.LogInformation("Photo storage path: {Path}", storageConfig?.BasePath ?? "./photos");
logger.LogInformation("Max file size: {Size}MB", storageConfig?.MaxFileSizeMB ?? 50);

await app.RunAsync();

// Make Program class accessible for integration tests
public partial class Program { }
