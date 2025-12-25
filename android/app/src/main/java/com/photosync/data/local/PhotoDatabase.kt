package com.photosync.data.local

import androidx.room.Database
import androidx.room.RoomDatabase

@Database(
    entities = [SyncedPhotoEntity::class],
    version = 1,
    exportSchema = true
)
abstract class PhotoDatabase : RoomDatabase() {
    abstract fun syncedPhotoDao(): SyncedPhotoDao

    companion object {
        const val DATABASE_NAME = "photo_sync_db"
    }
}
