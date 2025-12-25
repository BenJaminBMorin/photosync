using PhotoSync.Core.Entities;

namespace PhotoSync.Core.Interfaces;

/// <summary>
/// Repository for photo entity persistence.
/// </summary>
public interface IPhotoRepository
{
    /// <summary>
    /// Gets a photo by its unique identifier.
    /// </summary>
    Task<Photo?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets a photo by its file hash.
    /// </summary>
    Task<Photo?> GetByHashAsync(string fileHash, CancellationToken cancellationToken = default);

    /// <summary>
    /// Checks which hashes from the provided list already exist.
    /// </summary>
    Task<IReadOnlyList<string>> GetExistingHashesAsync(IEnumerable<string> hashes, CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets all photos with pagination.
    /// </summary>
    Task<IReadOnlyList<Photo>> GetAllAsync(int skip, int take, CancellationToken cancellationToken = default);

    /// <summary>
    /// Gets the total count of photos.
    /// </summary>
    Task<int> GetCountAsync(CancellationToken cancellationToken = default);

    /// <summary>
    /// Adds a new photo.
    /// </summary>
    Task AddAsync(Photo photo, CancellationToken cancellationToken = default);

    /// <summary>
    /// Deletes a photo by its identifier.
    /// </summary>
    Task<bool> DeleteAsync(Guid id, CancellationToken cancellationToken = default);

    /// <summary>
    /// Saves all pending changes.
    /// </summary>
    Task SaveChangesAsync(CancellationToken cancellationToken = default);
}
