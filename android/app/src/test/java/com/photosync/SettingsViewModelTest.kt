package com.photosync

import app.cash.turbine.test
import com.google.common.truth.Truth.assertThat
import com.photosync.data.repository.PhotoRepository
import com.photosync.data.repository.SettingsRepository
import com.photosync.domain.model.Settings
import com.photosync.ui.settings.SettingsViewModel
import com.photosync.ui.settings.TestResult
import io.mockk.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.*
import org.junit.After
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class SettingsViewModelTest {

    private lateinit var settingsRepository: SettingsRepository
    private lateinit var photoRepository: PhotoRepository
    private lateinit var viewModel: SettingsViewModel
    private val testDispatcher = StandardTestDispatcher()

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        settingsRepository = mockk(relaxed = true)
        photoRepository = mockk(relaxed = true)

        every { settingsRepository.settings } returns flowOf(
            Settings(serverUrl = "http://test.com", apiKey = "testkey", wifiOnly = true)
        )
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `initial state loads settings from repository`() = runTest {
        // Act
        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Assert
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.serverUrl).isEqualTo("http://test.com")
            assertThat(state.apiKey).isEqualTo("testkey")
            assertThat(state.wifiOnly).isTrue()
        }
    }

    @Test
    fun `updateServerUrl updates state`() = runTest {
        // Arrange
        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Act
        viewModel.updateServerUrl("http://new-server.com")

        // Assert
        assertThat(viewModel.uiState.value.serverUrl).isEqualTo("http://new-server.com")
    }

    @Test
    fun `updateApiKey updates state`() = runTest {
        // Arrange
        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Act
        viewModel.updateApiKey("new-api-key")

        // Assert
        assertThat(viewModel.uiState.value.apiKey).isEqualTo("new-api-key")
    }

    @Test
    fun `updateWifiOnly updates state and persists`() = runTest {
        // Arrange
        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Act
        viewModel.updateWifiOnly(false)
        advanceUntilIdle()

        // Assert
        assertThat(viewModel.uiState.value.wifiOnly).isFalse()
        coVerify { settingsRepository.setWifiOnly(false) }
    }

    @Test
    fun `testConnection success updates state with success result`() = runTest {
        // Arrange
        coEvery { photoRepository.testConnection() } returns Result.success(Unit)

        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Act
        viewModel.testConnection()
        advanceUntilIdle()

        // Assert
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.isTesting).isFalse()
            assertThat(state.testResult).isEqualTo(TestResult.Success)
        }
    }

    @Test
    fun `testConnection failure updates state with error result`() = runTest {
        // Arrange
        coEvery { photoRepository.testConnection() } returns Result.failure(Exception("Connection failed"))

        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        // Act
        viewModel.testConnection()
        advanceUntilIdle()

        // Assert
        viewModel.uiState.test {
            val state = awaitItem()
            assertThat(state.isTesting).isFalse()
            assertThat(state.testResult).isInstanceOf(TestResult.Error::class.java)
            assertThat((state.testResult as TestResult.Error).message).contains("Connection failed")
        }
    }

    @Test
    fun `saveSettings persists all settings`() = runTest {
        // Arrange
        viewModel = SettingsViewModel(settingsRepository, photoRepository)
        advanceUntilIdle()

        viewModel.updateServerUrl("http://new-server.com")
        viewModel.updateApiKey("new-key")

        // Act
        viewModel.saveSettings()
        advanceUntilIdle()

        // Assert
        coVerify {
            settingsRepository.saveSettings(
                match {
                    it.serverUrl == "http://new-server.com" && it.apiKey == "new-key"
                }
            )
        }
    }
}
