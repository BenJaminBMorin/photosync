using System.Text;
using FluentAssertions;
using PhotoSync.Infrastructure.Services;

namespace PhotoSync.Tests.Services;

public class HashServiceTests
{
    private readonly HashService _sut = new();

    [Fact]
    public async Task ComputeHashAsync_WithContent_ReturnsConsistentHash()
    {
        // Arrange
        var content = "Hello, World!"u8.ToArray();
        using var stream = new MemoryStream(content);

        // Act
        var hash = await _sut.ComputeHashAsync(stream);

        // Assert
        hash.Should().NotBeNullOrEmpty();
        hash.Should().HaveLength(64); // SHA256 = 64 hex chars
        hash.Should().MatchRegex("^[a-f0-9]{64}$");
    }

    [Fact]
    public async Task ComputeHashAsync_SameContent_ReturnsSameHash()
    {
        // Arrange
        var content = "Test content for hashing"u8.ToArray();
        using var stream1 = new MemoryStream(content);
        using var stream2 = new MemoryStream(content);

        // Act
        var hash1 = await _sut.ComputeHashAsync(stream1);
        var hash2 = await _sut.ComputeHashAsync(stream2);

        // Assert
        hash1.Should().Be(hash2);
    }

    [Fact]
    public async Task ComputeHashAsync_DifferentContent_ReturnsDifferentHash()
    {
        // Arrange
        using var stream1 = new MemoryStream("Content A"u8.ToArray());
        using var stream2 = new MemoryStream("Content B"u8.ToArray());

        // Act
        var hash1 = await _sut.ComputeHashAsync(stream1);
        var hash2 = await _sut.ComputeHashAsync(stream2);

        // Assert
        hash1.Should().NotBe(hash2);
    }

    [Fact]
    public async Task ComputeHashAsync_ResetsStreamPosition()
    {
        // Arrange
        var content = "Test content"u8.ToArray();
        using var stream = new MemoryStream(content);
        stream.Position = 5; // Move position forward

        // Act
        await _sut.ComputeHashAsync(stream);

        // Assert - position should be reset to where we started
        stream.Position.Should().Be(5);
    }

    [Theory]
    [InlineData("abc123def456abc123def456abc123def456abc123def456abc123def456abcd")]
    [InlineData("ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD")]
    [InlineData("sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abcd")]
    public void IsValidHash_ValidHash_ReturnsTrue(string hash)
    {
        // Act
        var result = _sut.IsValidHash(hash);

        // Assert
        result.Should().BeTrue();
    }

    [Theory]
    [InlineData("")]
    [InlineData("   ")]
    [InlineData("abc")] // too short
    [InlineData("xyz123")] // too short
    [InlineData("abc123def456abc123def456abc123def456abc123def456abc123def456abcZ")] // invalid char Z
    public void IsValidHash_InvalidHash_ReturnsFalse(string hash)
    {
        // Act
        var result = _sut.IsValidHash(hash);

        // Assert
        result.Should().BeFalse();
    }

    [Fact]
    public void NormalizeHash_WithPrefix_RemovesPrefix()
    {
        // Arrange
        var hash = "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abcd";

        // Act
        var result = _sut.NormalizeHash(hash);

        // Assert
        result.Should().Be("abc123def456abc123def456abc123def456abc123def456abc123def456abcd");
    }

    [Fact]
    public void NormalizeHash_UpperCase_ReturnsLowerCase()
    {
        // Arrange
        var hash = "ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD";

        // Act
        var result = _sut.NormalizeHash(hash);

        // Assert
        result.Should().Be("abc123def456abc123def456abc123def456abc123def456abc123def456abcd");
    }
}
