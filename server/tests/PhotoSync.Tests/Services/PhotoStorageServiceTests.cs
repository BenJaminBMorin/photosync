using FluentAssertions;
using PhotoSync.Infrastructure.Services;

namespace PhotoSync.Tests.Services;

public class PhotoStorageServiceTests : IDisposable
{
    private readonly string _testBasePath;
    private readonly PhotoStorageService _sut;

    public PhotoStorageServiceTests()
    {
        _testBasePath = Path.Combine(Path.GetTempPath(), $"PhotoSyncTests_{Guid.NewGuid():N}");
        Directory.CreateDirectory(_testBasePath);
        _sut = new PhotoStorageService(_testBasePath);
    }

    public void Dispose()
    {
        // Clean up test directory
        if (Directory.Exists(_testBasePath))
        {
            Directory.Delete(_testBasePath, recursive: true);
        }
    }

    [Fact]
    public async Task StoreAsync_ValidFile_StoresInYearMonthFolder()
    {
        // Arrange
        var content = "fake image content"u8.ToArray();
        using var stream = new MemoryStream(content);
        var filename = "test_photo.jpg";
        var dateTaken = new DateTime(2024, 3, 15);

        // Act
        var storedPath = await _sut.StoreAsync(stream, filename, dateTaken);

        // Assert
        storedPath.Should().StartWith("2024/03/");
        storedPath.Should().EndWith(".jpg");

        var fullPath = _sut.GetFullPath(storedPath);
        File.Exists(fullPath).Should().BeTrue();
    }

    [Fact]
    public async Task StoreAsync_DuplicateFilename_CreatesUniqueFilename()
    {
        // Arrange
        var content = "fake image content"u8.ToArray();
        var filename = "duplicate.jpg";
        var dateTaken = new DateTime(2024, 6, 20);

        // Act - store same filename twice
        using var stream1 = new MemoryStream(content);
        var path1 = await _sut.StoreAsync(stream1, filename, dateTaken);

        using var stream2 = new MemoryStream(content);
        var path2 = await _sut.StoreAsync(stream2, filename, dateTaken);

        // Assert
        path1.Should().NotBe(path2);
        _sut.Exists(path1).Should().BeTrue();
        _sut.Exists(path2).Should().BeTrue();
    }

    [Theory]
    [InlineData(".exe")]
    [InlineData(".bat")]
    [InlineData(".sh")]
    [InlineData(".php")]
    public async Task StoreAsync_DisallowedExtension_ThrowsException(string extension)
    {
        // Arrange
        var content = "malicious content"u8.ToArray();
        using var stream = new MemoryStream(content);
        var filename = $"file{extension}";

        // Act
        var act = () => _sut.StoreAsync(stream, filename, DateTime.UtcNow);

        // Assert
        await act.Should().ThrowAsync<InvalidOperationException>()
            .WithMessage("*not allowed*");
    }

    [Theory]
    [InlineData("../../../etc/passwd")]
    [InlineData("..\\..\\windows\\system32\\config")]
    [InlineData("/etc/passwd")]
    public async Task StoreAsync_PathTraversal_SanitizesFilename(string maliciousPath)
    {
        // Arrange
        var content = "content"u8.ToArray();
        using var stream = new MemoryStream(content);
        // Add .jpg extension to pass extension check
        var filename = maliciousPath + ".jpg";

        // Act
        var storedPath = await _sut.StoreAsync(stream, filename, DateTime.UtcNow);

        // Assert
        storedPath.Should().NotContain("..");
        storedPath.Should().NotContain("/etc/");
        storedPath.Should().NotContain("\\windows\\");

        var fullPath = _sut.GetFullPath(storedPath);
        fullPath.Should().StartWith(_testBasePath);
    }

    [Fact]
    public void GetFullPath_PathTraversal_ThrowsException()
    {
        // Arrange
        var maliciousPath = "../../../etc/passwd";

        // Act
        var act = () => _sut.GetFullPath(maliciousPath);

        // Assert
        act.Should().Throw<InvalidOperationException>()
            .WithMessage("*path traversal*");
    }

    [Fact]
    public async Task Delete_ExistingFile_ReturnsTrue()
    {
        // Arrange
        var content = "content to delete"u8.ToArray();
        using var stream = new MemoryStream(content);
        var storedPath = await _sut.StoreAsync(stream, "delete_me.jpg", DateTime.UtcNow);

        // Act
        var result = _sut.Delete(storedPath);

        // Assert
        result.Should().BeTrue();
        _sut.Exists(storedPath).Should().BeFalse();
    }

    [Fact]
    public void Delete_NonExistentFile_ReturnsFalse()
    {
        // Act
        var result = _sut.Delete("2024/01/nonexistent.jpg");

        // Assert
        result.Should().BeFalse();
    }

    [Fact]
    public void Exists_ExistingFile_ReturnsTrue()
    {
        // Arrange
        var yearFolder = Path.Combine(_testBasePath, "2024", "01");
        Directory.CreateDirectory(yearFolder);
        var testFile = Path.Combine(yearFolder, "test.jpg");
        File.WriteAllText(testFile, "test");

        // Act
        var result = _sut.Exists("2024/01/test.jpg");

        // Assert
        result.Should().BeTrue();
    }

    [Fact]
    public void Exists_NonExistentFile_ReturnsFalse()
    {
        // Act
        var result = _sut.Exists("2024/01/nonexistent.jpg");

        // Assert
        result.Should().BeFalse();
    }
}
