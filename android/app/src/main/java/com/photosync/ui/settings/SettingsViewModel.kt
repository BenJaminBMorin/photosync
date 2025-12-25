package com.photosync.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.photosync.data.repository.PhotoRepository
import com.photosync.data.repository.SettingsRepository
import com.photosync.domain.model.Settings
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SettingsUiState(
    val serverUrl: String = "",
    val apiKey: String = "",
    val wifiOnly: Boolean = true,
    val isTesting: Boolean = false,
    val testResult: TestResult? = null,
    val isSaving: Boolean = false
)

sealed class TestResult {
    data object Success : TestResult()
    data class Error(val message: String) : TestResult()
}

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val settingsRepository: SettingsRepository,
    private val photoRepository: PhotoRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    init {
        loadSettings()
    }

    private fun loadSettings() {
        viewModelScope.launch {
            settingsRepository.settings.collect { settings ->
                _uiState.update {
                    it.copy(
                        serverUrl = settings.serverUrl,
                        apiKey = settings.apiKey,
                        wifiOnly = settings.wifiOnly
                    )
                }
            }
        }
    }

    fun updateServerUrl(url: String) {
        _uiState.update { it.copy(serverUrl = url, testResult = null) }
    }

    fun updateApiKey(apiKey: String) {
        _uiState.update { it.copy(apiKey = apiKey, testResult = null) }
    }

    fun updateWifiOnly(wifiOnly: Boolean) {
        _uiState.update { it.copy(wifiOnly = wifiOnly) }
        viewModelScope.launch {
            settingsRepository.setWifiOnly(wifiOnly)
        }
    }

    fun testConnection() {
        viewModelScope.launch {
            _uiState.update { it.copy(isTesting = true, testResult = null) }

            // First save the current settings
            settingsRepository.saveSettings(
                Settings(
                    serverUrl = _uiState.value.serverUrl,
                    apiKey = _uiState.value.apiKey,
                    wifiOnly = _uiState.value.wifiOnly
                )
            )

            // Then test connection
            val result = photoRepository.testConnection()

            _uiState.update {
                it.copy(
                    isTesting = false,
                    testResult = if (result.isSuccess) {
                        TestResult.Success
                    } else {
                        TestResult.Error(result.exceptionOrNull()?.message ?: "Unknown error")
                    }
                )
            }
        }
    }

    fun saveSettings() {
        viewModelScope.launch {
            _uiState.update { it.copy(isSaving = true) }

            settingsRepository.saveSettings(
                Settings(
                    serverUrl = _uiState.value.serverUrl,
                    apiKey = _uiState.value.apiKey,
                    wifiOnly = _uiState.value.wifiOnly
                )
            )

            _uiState.update { it.copy(isSaving = false) }
        }
    }

    fun clearTestResult() {
        _uiState.update { it.copy(testResult = null) }
    }
}
