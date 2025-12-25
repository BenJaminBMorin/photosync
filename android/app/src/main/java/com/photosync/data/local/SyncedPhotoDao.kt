package com.photosync.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import kotlinx.coroutines.flow.Flow

@Dao
interface SyncedPhotoDao {

    /**
     * Get all synced photo paths as a Flow for reactive updates.
     */
    @Query("SELECT devicePath FROM synced_photos")
    fun getAllSyncedPaths(): Flow<List<String>>

    /**
     * Get all synced photo paths (non-reactive).
     */
    @Query("SELECT devicePath FROM synced_photos")
    suspend fun getAllSyncedPathsOnce(): List<String>

    /**
     * Check if a photo at the given path has been synced.
     */
    @Query("SELECT EXISTS(SELECT 1 FROM synced_photos WHERE devicePath = :path)")
    suspend fun isSynced(path: String): Boolean

    /**
     * Check multiple paths at once for sync status.
     */
    @Query("SELECT devicePath FROM synced_photos WHERE devicePath IN (:paths)")
    suspend fun getSyncedPathsFromList(paths: List<String>): List<String>

    /**
     * Insert a synced photo record.
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(syncedPhoto: SyncedPhotoEntity)

    /**
     * Insert multiple synced photo records.
     */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(syncedPhotos: List<SyncedPhotoEntity>)

    /**
     * Delete a synced photo record by path.
     */
    @Query("DELETE FROM synced_photos WHERE devicePath = :path")
    suspend fun deleteByPath(path: String)

    /**
     * Get count of synced photos.
     */
    @Query("SELECT COUNT(*) FROM synced_photos")
    suspend fun getCount(): Int

    /**
     * Clear all sync records (for debugging/reset).
     */
    @Query("DELETE FROM synced_photos")
    suspend fun clearAll()
}
