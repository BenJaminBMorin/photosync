using Microsoft.AspNetCore.Mvc;
using PhotoSync.Core.DTOs;
using PhotoSync.Core.Entities;
using PhotoSync.Core.Interfaces;

namespace PhotoSync.Api.Endpoints;

/// <summary>
/// Photo API endpoint definitions.
/// </summary>
public static class PhotoEndpoints
{
    public static void MapPhotoEndpoints(this IEndpointRouteBuilder routes)
    {
        var group = routes.MapGroup("/api/photos")
            .WithTags("Photos");

        group.MapPost("/upload", UploadPhotoAsync)
            .DisableAntiforgery() // Required for file uploads
            .WithName("UploadPhoto")
            .WithDescription("Upload a single photo")
            .Produces<UploadResult>(StatusCodes.Status200OK)
            .Produces(StatusCodes.Status400BadRequest)
            .Produces(StatusCodes.Status401Unauthorized);

        group.MapPost("/check", CheckHashesAsync)
            .WithName("CheckHashes")
            .WithDescription("Check which file hashes already exist on the server")
            .Produces<CheckHashesResult>(StatusCodes.Status200OK)
            .Produces(StatusCodes.Status400BadRequest);

        group.MapGet("/", GetPhotosAsync)
            .WithName("GetPhotos")
            .WithDescription("Get all photos with pagination")
            .Produces<PhotoListResponse>(StatusCodes.Status200OK);

        group.MapGet("/{id:guid}", GetPhotoByIdAsync)
            .WithName("GetPhotoById")
            .WithDescription("Get a photo by ID")
            .Produces<PhotoResponse>(StatusCodes.Status200OK)
            .Produces(StatusCodes.Status404NotFound);

        group.MapDelete("/{id:guid}", DeletePhotoAsync)
            .WithName("DeletePhoto")
            .WithDescription("Delete a photo by ID")
            .Produces(StatusCodes.Status204NoContent)
            .Produces(StatusCodes.Status404NotFound);
    }

    private static async Task<IResult> UploadPhotoAsync(
        HttpRequest request,
        [FromServices] IPhotoRepository repository,
        [FromServices] IPhotoStorageService storageService,
        [FromServices] IHashService hashService,
        [FromServices] ILogger<Program> logger,
        CancellationToken cancellationToken)
    {
        // Validate content type
        if (!request.HasFormContentType)
        {
            return Results.BadRequest(new { error = "Request must be multipart/form-data." });
        }

        var form = await request.ReadFormAsync(cancellationToken);
        var file = form.Files.GetFile("file");

        if (file is null || file.Length == 0)
        {
            return Results.BadRequest(new { error = "No file provided or file is empty." });
        }

        // Get metadata from form
        var originalFilename = form["originalFilename"].FirstOrDefault() ?? file.FileName;
        var dateTakenStr = form["dateTaken"].FirstOrDefault();

        if (!DateTime.TryParse(dateTakenStr, out var dateTaken))
        {
            dateTaken = DateTime.UtcNow; // Default to now if not provided
        }

        try
        {
            // Compute hash for duplicate detection
            await using var stream = file.OpenReadStream();
            var fileHash = await hashService.ComputeHashAsync(stream, cancellationToken);

            // Check for duplicate
            var existingPhoto = await repository.GetByHashAsync(fileHash, cancellationToken);
            if (existingPhoto is not null)
            {
                logger.LogInformation("Duplicate photo detected: {Hash}", fileHash);
                return Results.Ok(UploadResult.Duplicate(
                    existingPhoto.Id,
                    existingPhoto.StoredPath,
                    existingPhoto.UploadedAt));
            }

            // Reset stream position for storage
            stream.Position = 0;

            // Store the file
            var storedPath = await storageService.StoreAsync(
                stream,
                originalFilename,
                dateTaken,
                cancellationToken);

            // Create database record
            var photo = Photo.Create(
                originalFilename,
                storedPath,
                fileHash,
                file.Length,
                dateTaken);

            await repository.AddAsync(photo, cancellationToken);
            await repository.SaveChangesAsync(cancellationToken);

            logger.LogInformation("Photo uploaded: {Id} -> {Path}", photo.Id, storedPath);

            return Results.Ok(UploadResult.NewUpload(photo.Id, storedPath, photo.UploadedAt));
        }
        catch (InvalidOperationException ex)
        {
            logger.LogWarning(ex, "Invalid upload attempt");
            return Results.BadRequest(new { error = ex.Message });
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error uploading photo");
            return Results.Problem("An error occurred while uploading the photo.");
        }
    }

    private static async Task<IResult> CheckHashesAsync(
        [FromBody] CheckHashesRequest request,
        [FromServices] IPhotoRepository repository,
        CancellationToken cancellationToken)
    {
        if (request.Hashes is null || request.Hashes.Count == 0)
        {
            return Results.BadRequest(new { error = "At least one hash is required." });
        }

        // Limit the number of hashes that can be checked at once
        const int maxHashes = 1000;
        if (request.Hashes.Count > maxHashes)
        {
            return Results.BadRequest(new { error = $"Maximum {maxHashes} hashes can be checked at once." });
        }

        var normalizedHashes = request.Hashes
            .Select(h => h.ToLowerInvariant().Trim())
            .Distinct()
            .ToList();

        var existing = await repository.GetExistingHashesAsync(normalizedHashes, cancellationToken);
        var existingSet = existing.ToHashSet();
        var missing = normalizedHashes.Where(h => !existingSet.Contains(h)).ToList();

        return Results.Ok(new CheckHashesResult
        {
            Existing = existing,
            Missing = missing
        });
    }

    private static async Task<IResult> GetPhotosAsync(
        [FromQuery] int skip = 0,
        [FromQuery] int take = 50,
        [FromServices] IPhotoRepository repository = null!,
        CancellationToken cancellationToken = default)
    {
        // Validate pagination
        skip = Math.Max(0, skip);
        take = Math.Clamp(take, 1, 100);

        var photos = await repository.GetAllAsync(skip, take, cancellationToken);
        var totalCount = await repository.GetCountAsync(cancellationToken);

        return Results.Ok(new PhotoListResponse
        {
            Photos = photos.Select(p => new PhotoResponse
            {
                Id = p.Id,
                OriginalFilename = p.OriginalFilename,
                StoredPath = p.StoredPath,
                FileSize = p.FileSize,
                DateTaken = p.DateTaken,
                UploadedAt = p.UploadedAt
            }).ToList(),
            TotalCount = totalCount,
            Skip = skip,
            Take = take
        });
    }

    private static async Task<IResult> GetPhotoByIdAsync(
        Guid id,
        [FromServices] IPhotoRepository repository,
        CancellationToken cancellationToken)
    {
        var photo = await repository.GetByIdAsync(id, cancellationToken);

        if (photo is null)
        {
            return Results.NotFound(new { error = "Photo not found." });
        }

        return Results.Ok(new PhotoResponse
        {
            Id = photo.Id,
            OriginalFilename = photo.OriginalFilename,
            StoredPath = photo.StoredPath,
            FileSize = photo.FileSize,
            DateTaken = photo.DateTaken,
            UploadedAt = photo.UploadedAt
        });
    }

    private static async Task<IResult> DeletePhotoAsync(
        Guid id,
        [FromServices] IPhotoRepository repository,
        [FromServices] IPhotoStorageService storageService,
        [FromServices] ILogger<Program> logger,
        CancellationToken cancellationToken)
    {
        var photo = await repository.GetByIdAsync(id, cancellationToken);

        if (photo is null)
        {
            return Results.NotFound(new { error = "Photo not found." });
        }

        // Delete file from storage
        storageService.Delete(photo.StoredPath);

        // Delete from database
        await repository.DeleteAsync(id, cancellationToken);
        await repository.SaveChangesAsync(cancellationToken);

        logger.LogInformation("Photo deleted: {Id}", id);

        return Results.NoContent();
    }
}

// Request/Response DTOs for the API
public sealed record CheckHashesRequest
{
    public required IReadOnlyList<string> Hashes { get; init; }
}

public sealed record PhotoResponse
{
    public required Guid Id { get; init; }
    public required string OriginalFilename { get; init; }
    public required string StoredPath { get; init; }
    public required long FileSize { get; init; }
    public required DateTime DateTaken { get; init; }
    public required DateTime UploadedAt { get; init; }
}

public sealed record PhotoListResponse
{
    public required IReadOnlyList<PhotoResponse> Photos { get; init; }
    public required int TotalCount { get; init; }
    public required int Skip { get; init; }
    public required int Take { get; init; }
}
