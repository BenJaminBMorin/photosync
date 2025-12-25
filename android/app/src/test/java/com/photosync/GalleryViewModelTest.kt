package com.photosync

import android.net.Uri
import app.cash.turbine.test
import com.google.common.truth.Truth.assertThat
import com.photosync.data.remote.UploadResponse
import com.photosync.data.repository.PhotoRepository
import com.photosync.data.repository.SettingsRepository
import com.photosync.domain.model.Photo
import com.photosync.domain.model.Settings
import com.photosync.ui.gallery.GalleryViewModel
import io.mockk.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.*
import org.junit.After
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class GalleryViewModelTest {

    private lateinit var photoRepository: PhotoRepository
    private lateinit var settingsRepository: SettingsRepository
    private lateinit var viewModel: GalleryViewModel
    private val testDispatcher = StandardTestDispatcher()

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        photoRepository = mockk(relaxed = true)
        settingsRepository = mockk(relaxed = true)

        every { settingsRepository.settings } returns flowOf(
            Settings(serverUrl = "http://test.com", apiKey = "testkey", wifiOnly = true)
        )
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `initial state shows loading then photos`() = runTest {
        // Arrange
        val mockUri = mockk<Uri>()
        val photos = listOf(
            Photo(id = 1, uri = mockUri, path = "/test/1.jpg", displayName = "1.jpg",
                dateTaken = System.currentTimeMillis(), size = 1000, isSynced = false),
            Photo(id = 2, uri = mockUri, path = "/test/2.jpg", displayName = "2.jpg",
                dateTaken = System.currentTimeMillis(), size = 2000, isSynced = true)
        )
        coEvery { photoRepository.getPhotosWithSyncStatus() } returns photos

        // Act
        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        // Assert
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.photos).hasSize(2)
            assertThat(state.syncedCount).isEqualTo(1)
            assertThat(state.unsyncedCount).isEqualTo(1)
            assertThat(state.isLoading).isFalse()
        }
    }

    @Test
    fun `togglePhotoSelection adds and removes from selection`() = runTest {
        // Arrange
        val mockUri = mockk<Uri>()
        val photos = listOf(
            Photo(id = 1, uri = mockUri, path = "/test/1.jpg", displayName = "1.jpg",
                dateTaken = System.currentTimeMillis(), size = 1000, isSynced = false)
        )
        coEvery { photoRepository.getPhotosWithSyncStatus() } returns photos

        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        // Act - select
        viewModel.togglePhotoSelection(1)

        // Assert - selected
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.selectedIds).contains(1L)
            assertThat(state.selectedCount).isEqualTo(1)
        }

        // Act - deselect
        viewModel.togglePhotoSelection(1)

        // Assert - deselected
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.selectedIds).isEmpty()
        }
    }

    @Test
    fun `selectAll selects only unsynced photos`() = runTest {
        // Arrange
        val mockUri = mockk<Uri>()
        val photos = listOf(
            Photo(id = 1, uri = mockUri, path = "/test/1.jpg", displayName = "1.jpg",
                dateTaken = System.currentTimeMillis(), size = 1000, isSynced = false),
            Photo(id = 2, uri = mockUri, path = "/test/2.jpg", displayName = "2.jpg",
                dateTaken = System.currentTimeMillis(), size = 2000, isSynced = true),
            Photo(id = 3, uri = mockUri, path = "/test/3.jpg", displayName = "3.jpg",
                dateTaken = System.currentTimeMillis(), size = 3000, isSynced = false)
        )
        coEvery { photoRepository.getPhotosWithSyncStatus() } returns photos

        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        // Act
        viewModel.selectAll()

        // Assert - only unsynced photos selected
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.selectedIds).containsExactly(1L, 3L)
            assertThat(state.selectedCount).isEqualTo(2)
        }
    }

    @Test
    fun `clearSelection removes all selections`() = runTest {
        // Arrange
        val mockUri = mockk<Uri>()
        val photos = listOf(
            Photo(id = 1, uri = mockUri, path = "/test/1.jpg", displayName = "1.jpg",
                dateTaken = System.currentTimeMillis(), size = 1000, isSynced = false)
        )
        coEvery { photoRepository.getPhotosWithSyncStatus() } returns photos

        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        viewModel.togglePhotoSelection(1)

        // Act
        viewModel.clearSelection()

        // Assert
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.selectedIds).isEmpty()
        }
    }

    @Test
    fun `toggleUnsyncedFilter changes filter state`() = runTest {
        // Arrange
        coEvery { photoRepository.getPhotosWithSyncStatus() } returns emptyList()

        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        // Assert initial
        assertThat(viewModel.uiState.value.showUnsyncedOnly).isFalse()

        // Act
        viewModel.toggleUnsyncedFilter()

        // Assert toggled
        assertThat(viewModel.uiState.value.showUnsyncedOnly).isTrue()
    }

    @Test
    fun `syncSelected uploads photos and updates state`() = runTest {
        // Arrange
        val mockUri = mockk<Uri>()
        val photo = Photo(id = 1, uri = mockUri, path = "/test/1.jpg", displayName = "1.jpg",
            dateTaken = System.currentTimeMillis(), size = 1000, isSynced = false)
        val syncedPhoto = photo.copy(isSynced = true)

        coEvery { photoRepository.getPhotosWithSyncStatus() } returnsMany listOf(
            listOf(photo),
            listOf(syncedPhoto)
        )
        coEvery { photoRepository.uploadPhoto(any()) } returns Result.success(
            UploadResponse(
                id = "server-id",
                storedPath = "2024/01/1.jpg",
                uploadedAt = "2024-01-01T00:00:00Z",
                isDuplicate = false
            )
        )

        viewModel = GalleryViewModel(photoRepository, settingsRepository)
        advanceUntilIdle()

        viewModel.togglePhotoSelection(1)

        // Act
        viewModel.syncSelected()
        advanceUntilIdle()

        // Assert
        coVerify { photoRepository.uploadPhoto(photo) }
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.isSyncing).isFalse()
            assertThat(state.selectedIds).isEmpty()
        }
    }
}
