package com.photosync.ui.gallery

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.photosync.data.repository.PhotoRepository
import com.photosync.data.repository.SettingsRepository
import com.photosync.domain.model.Photo
import com.photosync.domain.model.SyncProgress
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class GalleryUiState(
    val photos: List<Photo> = emptyList(),
    val selectedIds: Set<Long> = emptySet(),
    val isLoading: Boolean = false,
    val isSyncing: Boolean = false,
    val syncProgress: SyncProgress? = null,
    val error: String? = null,
    val showUnsyncedOnly: Boolean = false,
    val isConfigured: Boolean = false
) {
    val selectedCount: Int get() = selectedIds.size

    val displayedPhotos: List<Photo>
        get() = if (showUnsyncedOnly) photos.filter { !it.isSynced } else photos

    val unsyncedCount: Int
        get() = photos.count { !it.isSynced }

    val syncedCount: Int
        get() = photos.count { it.isSynced }
}

sealed class GalleryEvent {
    data object PhotosLoaded : GalleryEvent()
    data object SyncCompleted : GalleryEvent()
    data class SyncError(val message: String) : GalleryEvent()
}

@HiltViewModel
class GalleryViewModel @Inject constructor(
    private val photoRepository: PhotoRepository,
    private val settingsRepository: SettingsRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(GalleryUiState())
    val uiState: StateFlow<GalleryUiState> = _uiState.asStateFlow()

    private val _events = MutableSharedFlow<GalleryEvent>()
    val events: SharedFlow<GalleryEvent> = _events.asSharedFlow()

    private var syncJob: Job? = null

    init {
        loadPhotos()
        observeSettings()
    }

    private fun observeSettings() {
        viewModelScope.launch {
            settingsRepository.settings.collect { settings ->
                _uiState.update { it.copy(isConfigured = settings.isConfigured) }
            }
        }
    }

    fun loadPhotos() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            try {
                val photos = photoRepository.getPhotosWithSyncStatus()
                _uiState.update {
                    it.copy(
                        photos = photos,
                        isLoading = false
                    )
                }
                _events.emit(GalleryEvent.PhotosLoaded)
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = "Failed to load photos: ${e.message}"
                    )
                }
            }
        }
    }

    fun togglePhotoSelection(photoId: Long) {
        _uiState.update { state ->
            val newSelection = if (photoId in state.selectedIds) {
                state.selectedIds - photoId
            } else {
                state.selectedIds + photoId
            }
            state.copy(selectedIds = newSelection)
        }
    }

    fun selectAll() {
        _uiState.update { state ->
            val idsToSelect = state.displayedPhotos
                .filter { !it.isSynced }
                .map { it.id }
                .toSet()
            state.copy(selectedIds = idsToSelect)
        }
    }

    fun clearSelection() {
        _uiState.update { it.copy(selectedIds = emptySet()) }
    }

    fun toggleUnsyncedFilter() {
        _uiState.update { it.copy(showUnsyncedOnly = !it.showUnsyncedOnly) }
    }

    fun syncSelected() {
        val selectedPhotos = _uiState.value.photos.filter { it.id in _uiState.value.selectedIds }
        if (selectedPhotos.isEmpty()) return

        syncJob = viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isSyncing = true,
                    syncProgress = SyncProgress(
                        total = selectedPhotos.size,
                        completed = 0,
                        currentFileName = null
                    )
                )
            }

            var completed = 0
            var failed = 0

            for (photo in selectedPhotos) {
                if (syncJob?.isActive != true) {
                    _uiState.update {
                        it.copy(
                            syncProgress = it.syncProgress?.copy(isCancelled = true)
                        )
                    }
                    break
                }

                _uiState.update {
                    it.copy(
                        syncProgress = SyncProgress(
                            total = selectedPhotos.size,
                            completed = completed,
                            currentFileName = photo.displayName,
                            failed = failed
                        )
                    )
                }

                val result = photoRepository.uploadPhoto(photo)

                if (result.isSuccess) {
                    completed++
                } else {
                    failed++
                }
            }

            // Reload photos to get updated sync status
            val updatedPhotos = photoRepository.getPhotosWithSyncStatus()

            _uiState.update {
                it.copy(
                    photos = updatedPhotos,
                    selectedIds = emptySet(),
                    isSyncing = false,
                    syncProgress = null
                )
            }

            if (failed > 0) {
                _events.emit(GalleryEvent.SyncError("$failed photos failed to sync"))
            } else {
                _events.emit(GalleryEvent.SyncCompleted)
            }
        }
    }

    fun cancelSync() {
        syncJob?.cancel()
        _uiState.update {
            it.copy(
                isSyncing = false,
                syncProgress = null
            )
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
