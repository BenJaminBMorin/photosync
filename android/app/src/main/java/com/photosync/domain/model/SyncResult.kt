package com.photosync.domain.model

/**
 * Result of uploading a photo to the server.
 */
data class UploadResult(
    val id: String,
    val storedPath: String,
    val uploadedAt: String,
    val isDuplicate: Boolean
)

/**
 * Result of syncing multiple photos.
 */
data class SyncProgress(
    val total: Int,
    val completed: Int,
    val currentFileName: String?,
    val failed: Int = 0,
    val isCancelled: Boolean = false
) {
    val progressPercent: Float
        get() = if (total > 0) completed.toFloat() / total else 0f

    val isComplete: Boolean
        get() = completed + failed >= total
}
