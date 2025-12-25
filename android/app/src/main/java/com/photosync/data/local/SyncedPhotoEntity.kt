package com.photosync.data.local

import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Room entity for tracking synced photos.
 * Once a photo is in this table, it's considered synced.
 */
@Entity(
    tableName = "synced_photos",
    indices = [
        Index(value = ["devicePath"], unique = true)
    ]
)
data class SyncedPhotoEntity(
    @PrimaryKey(autoGenerate = true)
    val id: Long = 0,

    /** The full path on the device (used as unique identifier) */
    val devicePath: String,

    /** Original filename */
    val displayName: String,

    /** File size in bytes */
    val fileSize: Long,

    /** Date the photo was taken (epoch millis) */
    val dateTaken: Long,

    /** When the sync completed (epoch millis) */
    val syncedAt: Long,

    /** ID returned from the server */
    val serverPhotoId: String
)
