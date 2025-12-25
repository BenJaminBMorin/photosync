using FluentAssertions;
using Microsoft.EntityFrameworkCore;
using PhotoSync.Core.Entities;
using PhotoSync.Infrastructure.Data;

namespace PhotoSync.Tests.Data;

public class PhotoRepositoryTests : IDisposable
{
    private readonly PhotoDbContext _context;
    private readonly PhotoRepository _sut;

    public PhotoRepositoryTests()
    {
        var options = new DbContextOptionsBuilder<PhotoDbContext>()
            .UseInMemoryDatabase(databaseName: $"TestDb_{Guid.NewGuid()}")
            .Options;

        _context = new PhotoDbContext(options);
        _sut = new PhotoRepository(_context);
    }

    public void Dispose()
    {
        _context.Dispose();
    }

    [Fact]
    public async Task AddAsync_ValidPhoto_AddsToDatabase()
    {
        // Arrange
        var photo = Photo.Create(
            "test.jpg",
            "2024/03/test.jpg",
            "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
            1024,
            DateTime.UtcNow);

        // Act
        await _sut.AddAsync(photo);
        await _sut.SaveChangesAsync();

        // Assert
        var count = await _sut.GetCountAsync();
        count.Should().Be(1);
    }

    [Fact]
    public async Task GetByIdAsync_ExistingPhoto_ReturnsPhoto()
    {
        // Arrange
        var photo = Photo.Create("test.jpg", "2024/03/test.jpg", "hash123", 1024, DateTime.UtcNow);
        await _sut.AddAsync(photo);
        await _sut.SaveChangesAsync();

        // Act
        var result = await _sut.GetByIdAsync(photo.Id);

        // Assert
        result.Should().NotBeNull();
        result!.Id.Should().Be(photo.Id);
        result.OriginalFilename.Should().Be("test.jpg");
    }

    [Fact]
    public async Task GetByIdAsync_NonExistentId_ReturnsNull()
    {
        // Act
        var result = await _sut.GetByIdAsync(Guid.NewGuid());

        // Assert
        result.Should().BeNull();
    }

    [Fact]
    public async Task GetByHashAsync_ExistingHash_ReturnsPhoto()
    {
        // Arrange
        var hash = "abc123def456abc123def456abc123def456abc123def456abc123def456abcd";
        var photo = Photo.Create("test.jpg", "2024/03/test.jpg", hash, 1024, DateTime.UtcNow);
        await _sut.AddAsync(photo);
        await _sut.SaveChangesAsync();

        // Act
        var result = await _sut.GetByHashAsync(hash);

        // Assert
        result.Should().NotBeNull();
        result!.FileHash.Should().Be(hash);
    }

    [Fact]
    public async Task GetByHashAsync_CaseInsensitive()
    {
        // Arrange
        var hash = "abc123def456abc123def456abc123def456abc123def456abc123def456abcd";
        var photo = Photo.Create("test.jpg", "2024/03/test.jpg", hash, 1024, DateTime.UtcNow);
        await _sut.AddAsync(photo);
        await _sut.SaveChangesAsync();

        // Act - search with uppercase
        var result = await _sut.GetByHashAsync(hash.ToUpperInvariant());

        // Assert
        result.Should().NotBeNull();
    }

    [Fact]
    public async Task GetExistingHashesAsync_WithMixedHashes_ReturnsOnlyExisting()
    {
        // Arrange
        var existingHash1 = "hash1hash1hash1hash1hash1hash1hash1hash1hash1hash1hash1hash11";
        var existingHash2 = "hash2hash2hash2hash2hash2hash2hash2hash2hash2hash2hash2hash22";
        var missingHash = "hash3hash3hash3hash3hash3hash3hash3hash3hash3hash3hash3hash33";

        await _sut.AddAsync(Photo.Create("a.jpg", "2024/01/a.jpg", existingHash1, 100, DateTime.UtcNow));
        await _sut.AddAsync(Photo.Create("b.jpg", "2024/01/b.jpg", existingHash2, 100, DateTime.UtcNow));
        await _sut.SaveChangesAsync();

        // Act
        var result = await _sut.GetExistingHashesAsync([existingHash1, existingHash2, missingHash]);

        // Assert
        result.Should().HaveCount(2);
        result.Should().Contain(existingHash1);
        result.Should().Contain(existingHash2);
        result.Should().NotContain(missingHash);
    }

    [Fact]
    public async Task GetAllAsync_WithPagination_ReturnsCorrectPage()
    {
        // Arrange
        for (var i = 0; i < 10; i++)
        {
            var photo = Photo.Create(
                $"photo{i}.jpg",
                $"2024/01/photo{i}.jpg",
                $"hash{i}hash{i}hash{i}hash{i}hash{i}hash{i}hash{i}hash{i}hash{i}hash{i}h{i:D2}",
                100,
                DateTime.UtcNow.AddDays(-i));
            await _sut.AddAsync(photo);
        }
        await _sut.SaveChangesAsync();

        // Act
        var page1 = await _sut.GetAllAsync(skip: 0, take: 3);
        var page2 = await _sut.GetAllAsync(skip: 3, take: 3);

        // Assert
        page1.Should().HaveCount(3);
        page2.Should().HaveCount(3);
        page1.Should().NotIntersectWith(page2);
    }

    [Fact]
    public async Task DeleteAsync_ExistingPhoto_ReturnsTrue()
    {
        // Arrange
        var photo = Photo.Create("delete.jpg", "2024/01/delete.jpg", "delhash", 100, DateTime.UtcNow);
        await _sut.AddAsync(photo);
        await _sut.SaveChangesAsync();

        // Act
        var result = await _sut.DeleteAsync(photo.Id);
        await _sut.SaveChangesAsync();

        // Assert
        result.Should().BeTrue();
        var count = await _sut.GetCountAsync();
        count.Should().Be(0);
    }

    [Fact]
    public async Task DeleteAsync_NonExistentPhoto_ReturnsFalse()
    {
        // Act
        var result = await _sut.DeleteAsync(Guid.NewGuid());

        // Assert
        result.Should().BeFalse();
    }
}
