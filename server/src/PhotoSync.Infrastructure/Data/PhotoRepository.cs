using Microsoft.EntityFrameworkCore;
using PhotoSync.Core.Entities;
using PhotoSync.Core.Interfaces;

namespace PhotoSync.Infrastructure.Data;

/// <summary>
/// EF Core implementation of the photo repository.
/// </summary>
public sealed class PhotoRepository : IPhotoRepository
{
    private readonly PhotoDbContext _context;

    public PhotoRepository(PhotoDbContext context)
    {
        _context = context ?? throw new ArgumentNullException(nameof(context));
    }

    public async Task<Photo?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.Photos
            .AsNoTracking()
            .FirstOrDefaultAsync(p => p.Id == id, cancellationToken);
    }

    public async Task<Photo?> GetByHashAsync(string fileHash, CancellationToken cancellationToken = default)
    {
        var normalizedHash = fileHash.ToLowerInvariant();
        return await _context.Photos
            .AsNoTracking()
            .FirstOrDefaultAsync(p => p.FileHash == normalizedHash, cancellationToken);
    }

    public async Task<IReadOnlyList<string>> GetExistingHashesAsync(
        IEnumerable<string> hashes,
        CancellationToken cancellationToken = default)
    {
        var normalizedHashes = hashes.Select(h => h.ToLowerInvariant()).ToList();

        if (normalizedHashes.Count == 0)
            return Array.Empty<string>();

        return await _context.Photos
            .AsNoTracking()
            .Where(p => normalizedHashes.Contains(p.FileHash))
            .Select(p => p.FileHash)
            .ToListAsync(cancellationToken);
    }

    public async Task<IReadOnlyList<Photo>> GetAllAsync(
        int skip,
        int take,
        CancellationToken cancellationToken = default)
    {
        return await _context.Photos
            .AsNoTracking()
            .OrderByDescending(p => p.DateTaken)
            .Skip(skip)
            .Take(take)
            .ToListAsync(cancellationToken);
    }

    public async Task<int> GetCountAsync(CancellationToken cancellationToken = default)
    {
        return await _context.Photos.CountAsync(cancellationToken);
    }

    public async Task AddAsync(Photo photo, CancellationToken cancellationToken = default)
    {
        await _context.Photos.AddAsync(photo, cancellationToken);
    }

    public async Task<bool> DeleteAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var photo = await _context.Photos.FindAsync(new object[] { id }, cancellationToken);
        if (photo is null)
            return false;

        _context.Photos.Remove(photo);
        return true;
    }

    public async Task SaveChangesAsync(CancellationToken cancellationToken = default)
    {
        await _context.SaveChangesAsync(cancellationToken);
    }
}
