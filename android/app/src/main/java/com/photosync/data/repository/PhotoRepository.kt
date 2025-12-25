package com.photosync.data.repository

import android.content.ContentResolver
import android.content.ContentUris
import android.content.Context
import android.net.Uri
import android.provider.MediaStore
import com.photosync.data.local.SyncedPhotoDao
import com.photosync.data.local.SyncedPhotoEntity
import com.photosync.data.remote.PhotoSyncApi
import com.photosync.data.remote.UploadResponse
import com.photosync.domain.model.Photo
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File
import java.io.FileOutputStream
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PhotoRepository @Inject constructor(
    @ApplicationContext private val context: Context,
    private val syncedPhotoDao: SyncedPhotoDao,
    private val photoSyncApi: PhotoSyncApi
) {
    private val contentResolver: ContentResolver = context.contentResolver

    /**
     * Scan device for all photos.
     */
    suspend fun scanDevicePhotos(): List<Photo> = withContext(Dispatchers.IO) {
        val photos = mutableListOf<Photo>()

        val projection = arrayOf(
            MediaStore.Images.Media._ID,
            MediaStore.Images.Media.DISPLAY_NAME,
            MediaStore.Images.Media.DATA,
            MediaStore.Images.Media.DATE_TAKEN,
            MediaStore.Images.Media.SIZE
        )

        val sortOrder = "${MediaStore.Images.Media.DATE_TAKEN} DESC"

        contentResolver.query(
            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            projection,
            null,
            null,
            sortOrder
        )?.use { cursor ->
            val idColumn = cursor.getColumnIndexOrThrow(MediaStore.Images.Media._ID)
            val nameColumn = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DISPLAY_NAME)
            val pathColumn = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DATA)
            val dateColumn = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DATE_TAKEN)
            val sizeColumn = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.SIZE)

            while (cursor.moveToNext()) {
                val id = cursor.getLong(idColumn)
                val name = cursor.getString(nameColumn) ?: "unknown"
                val path = cursor.getString(pathColumn) ?: ""
                val dateTaken = cursor.getLong(dateColumn)
                val size = cursor.getLong(sizeColumn)

                val uri = ContentUris.withAppendedId(
                    MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
                    id
                )

                photos.add(
                    Photo(
                        id = id,
                        uri = uri,
                        path = path,
                        displayName = name,
                        dateTaken = dateTaken,
                        size = size
                    )
                )
            }
        }

        photos
    }

    /**
     * Get flow of synced photo paths for reactive UI updates.
     */
    fun getSyncedPathsFlow(): Flow<Set<String>> {
        return syncedPhotoDao.getAllSyncedPaths().map { it.toSet() }
    }

    /**
     * Get photos with their sync status.
     */
    suspend fun getPhotosWithSyncStatus(): List<Photo> = withContext(Dispatchers.IO) {
        val allPhotos = scanDevicePhotos()
        val syncedPaths = syncedPhotoDao.getAllSyncedPathsOnce().toSet()

        allPhotos.map { photo ->
            photo.copy(isSynced = syncedPaths.contains(photo.path))
        }
    }

    /**
     * Get only unsynced photos.
     */
    suspend fun getUnsyncedPhotos(): List<Photo> {
        return getPhotosWithSyncStatus().filter { !it.isSynced }
    }

    /**
     * Upload a single photo to the server.
     */
    suspend fun uploadPhoto(photo: Photo): Result<UploadResponse> = withContext(Dispatchers.IO) {
        try {
            // Create temp file from content URI
            val tempFile = createTempFileFromUri(photo.uri, photo.displayName)
                ?: return@withContext Result.failure(Exception("Could not read photo file"))

            try {
                val requestFile = tempFile.asRequestBody("image/*".toMediaTypeOrNull())
                val filePart = MultipartBody.Part.createFormData("file", photo.displayName, requestFile)

                val dateFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
                val dateTakenStr = dateFormat.format(Date(photo.dateTaken))

                val filenameBody = photo.displayName.toRequestBody("text/plain".toMediaTypeOrNull())
                val dateBody = dateTakenStr.toRequestBody("text/plain".toMediaTypeOrNull())

                val response = photoSyncApi.uploadPhoto(filePart, filenameBody, dateBody)

                if (response.isSuccessful && response.body() != null) {
                    val uploadResult = response.body()!!

                    // Mark as synced in local database
                    syncedPhotoDao.insert(
                        SyncedPhotoEntity(
                            devicePath = photo.path,
                            displayName = photo.displayName,
                            fileSize = photo.size,
                            dateTaken = photo.dateTaken,
                            syncedAt = System.currentTimeMillis(),
                            serverPhotoId = uploadResult.id
                        )
                    )

                    Result.success(uploadResult)
                } else {
                    val errorBody = response.errorBody()?.string() ?: "Unknown error"
                    Result.failure(Exception("Upload failed: $errorBody"))
                }
            } finally {
                tempFile.delete()
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Create a temporary file from a content URI.
     */
    private fun createTempFileFromUri(uri: Uri, filename: String): File? {
        return try {
            val extension = filename.substringAfterLast('.', "jpg")
            val tempFile = File.createTempFile("upload_", ".$extension", context.cacheDir)

            contentResolver.openInputStream(uri)?.use { input ->
                FileOutputStream(tempFile).use { output ->
                    input.copyTo(output)
                }
            }

            tempFile
        } catch (e: Exception) {
            null
        }
    }

    /**
     * Check if a specific photo is synced.
     */
    suspend fun isPhotoSynced(path: String): Boolean {
        return syncedPhotoDao.isSynced(path)
    }

    /**
     * Get count of synced photos.
     */
    suspend fun getSyncedCount(): Int {
        return syncedPhotoDao.getCount()
    }

    /**
     * Test connection to server.
     */
    suspend fun testConnection(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = photoSyncApi.healthCheck()
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Server returned ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
