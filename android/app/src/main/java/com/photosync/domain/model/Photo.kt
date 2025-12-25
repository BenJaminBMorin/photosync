package com.photosync.domain.model

import android.net.Uri

/**
 * Represents a photo on the device.
 */
data class Photo(
    val id: Long,
    val uri: Uri,
    val path: String,
    val displayName: String,
    val dateTaken: Long,
    val size: Long,
    val isSynced: Boolean = false,
    val syncedAt: Long? = null,
    val serverPhotoId: String? = null
)

/**
 * State of the sync operation for a photo.
 */
enum class SyncState {
    NOT_SYNCED,
    SYNCING,
    SYNCED,
    ERROR
}

/**
 * Photo with its current sync state for UI display.
 */
data class PhotoWithSyncState(
    val photo: Photo,
    val syncState: SyncState,
    val isSelected: Boolean = false
)
