using FluentAssertions;
using PhotoSync.Core.Entities;

namespace PhotoSync.Tests.Entities;

public class PhotoTests
{
    [Fact]
    public void Create_ValidParameters_ReturnsPhoto()
    {
        // Arrange
        var filename = "test_photo.jpg";
        var storedPath = "2024/03/test_photo.jpg";
        var hash = "abc123def456abc123def456abc123def456abc123def456abc123def456abcd";
        var fileSize = 1024L;
        var dateTaken = new DateTime(2024, 3, 15);

        // Act
        var photo = Photo.Create(filename, storedPath, hash, fileSize, dateTaken);

        // Assert
        photo.Should().NotBeNull();
        photo.Id.Should().NotBeEmpty();
        photo.OriginalFilename.Should().Be(filename);
        photo.StoredPath.Should().Be(storedPath);
        photo.FileHash.Should().Be(hash.ToLowerInvariant());
        photo.FileSize.Should().Be(fileSize);
        photo.DateTaken.Should().Be(dateTaken);
        photo.UploadedAt.Should().BeCloseTo(DateTime.UtcNow, TimeSpan.FromSeconds(5));
    }

    [Theory]
    [InlineData("")]
    [InlineData("   ")]
    [InlineData(null)]
    public void Create_EmptyFilename_ThrowsArgumentException(string? filename)
    {
        // Act
        var act = () => Photo.Create(
            filename!,
            "2024/03/test.jpg",
            "abc123",
            1024,
            DateTime.UtcNow);

        // Assert
        act.Should().Throw<ArgumentException>()
            .WithMessage("*filename*");
    }

    [Theory]
    [InlineData("")]
    [InlineData("   ")]
    [InlineData(null)]
    public void Create_EmptyStoredPath_ThrowsArgumentException(string? storedPath)
    {
        // Act
        var act = () => Photo.Create(
            "test.jpg",
            storedPath!,
            "abc123",
            1024,
            DateTime.UtcNow);

        // Assert
        act.Should().Throw<ArgumentException>()
            .WithMessage("*path*");
    }

    [Theory]
    [InlineData("")]
    [InlineData("   ")]
    [InlineData(null)]
    public void Create_EmptyHash_ThrowsArgumentException(string? hash)
    {
        // Act
        var act = () => Photo.Create(
            "test.jpg",
            "2024/03/test.jpg",
            hash!,
            1024,
            DateTime.UtcNow);

        // Assert
        act.Should().Throw<ArgumentException>()
            .WithMessage("*hash*");
    }

    [Theory]
    [InlineData(0)]
    [InlineData(-1)]
    [InlineData(-1000)]
    public void Create_InvalidFileSize_ThrowsArgumentException(long fileSize)
    {
        // Act
        var act = () => Photo.Create(
            "test.jpg",
            "2024/03/test.jpg",
            "abc123",
            fileSize,
            DateTime.UtcNow);

        // Assert
        act.Should().Throw<ArgumentException>()
            .WithMessage("*size*");
    }

    [Fact]
    public void Create_FilenameWithPath_SanitizesFilename()
    {
        // Arrange
        var maliciousFilename = "../../../etc/passwd.jpg";

        // Act
        var photo = Photo.Create(
            maliciousFilename,
            "2024/03/safe.jpg",
            "abc123",
            1024,
            DateTime.UtcNow);

        // Assert
        photo.OriginalFilename.Should().NotContain("..");
        photo.OriginalFilename.Should().NotContain("/");
        photo.OriginalFilename.Should().EndWith(".jpg");
    }

    [Fact]
    public void Create_HashWithUpperCase_NormalizesToLowerCase()
    {
        // Arrange
        var upperHash = "ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD";

        // Act
        var photo = Photo.Create(
            "test.jpg",
            "2024/03/test.jpg",
            upperHash,
            1024,
            DateTime.UtcNow);

        // Assert
        photo.FileHash.Should().Be(upperHash.ToLowerInvariant());
    }

    [Fact]
    public void Create_GeneratesUniqueId()
    {
        // Act
        var photo1 = Photo.Create("a.jpg", "2024/01/a.jpg", "hash1", 100, DateTime.UtcNow);
        var photo2 = Photo.Create("b.jpg", "2024/01/b.jpg", "hash2", 100, DateTime.UtcNow);

        // Assert
        photo1.Id.Should().NotBe(photo2.Id);
    }
}
