using Microsoft.EntityFrameworkCore;
using PhotoSync.Core.Entities;

namespace PhotoSync.Infrastructure.Data;

/// <summary>
/// Entity Framework Core database context for photo storage.
/// </summary>
public sealed class PhotoDbContext : DbContext
{
    public PhotoDbContext(DbContextOptions<PhotoDbContext> options)
        : base(options)
    {
    }

    public DbSet<Photo> Photos => Set<Photo>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);

        modelBuilder.Entity<Photo>(entity =>
        {
            entity.ToTable("photos");

            entity.HasKey(e => e.Id);

            entity.Property(e => e.Id)
                .HasColumnName("id")
                .ValueGeneratedNever(); // We generate GUIDs in the domain

            entity.Property(e => e.OriginalFilename)
                .HasColumnName("original_filename")
                .IsRequired()
                .HasMaxLength(500);

            entity.Property(e => e.StoredPath)
                .HasColumnName("stored_path")
                .IsRequired()
                .HasMaxLength(1000);

            entity.Property(e => e.FileHash)
                .HasColumnName("file_hash")
                .IsRequired()
                .HasMaxLength(64); // SHA256 hex = 64 chars

            entity.Property(e => e.FileSize)
                .HasColumnName("file_size")
                .IsRequired();

            entity.Property(e => e.DateTaken)
                .HasColumnName("date_taken")
                .IsRequired();

            entity.Property(e => e.UploadedAt)
                .HasColumnName("uploaded_at")
                .IsRequired();

            // Index on hash for duplicate detection
            entity.HasIndex(e => e.FileHash)
                .HasDatabaseName("idx_photos_hash");

            // Index on date for queries
            entity.HasIndex(e => e.DateTaken)
                .HasDatabaseName("idx_photos_date");
        });
    }
}
