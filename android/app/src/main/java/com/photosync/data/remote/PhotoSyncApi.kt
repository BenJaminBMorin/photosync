package com.photosync.data.remote

import okhttp3.MultipartBody
import okhttp3.RequestBody
import retrofit2.Response
import retrofit2.http.*

interface PhotoSyncApi {

    /**
     * Upload a single photo to the server.
     */
    @Multipart
    @POST("api/photos/upload")
    suspend fun uploadPhoto(
        @Part file: MultipartBody.Part,
        @Part("originalFilename") originalFilename: RequestBody,
        @Part("dateTaken") dateTaken: RequestBody
    ): Response<UploadResponse>

    /**
     * Check which hashes already exist on the server.
     */
    @POST("api/photos/check")
    suspend fun checkHashes(
        @Body request: CheckHashesRequest
    ): Response<CheckHashesResponse>

    /**
     * Get list of photos on server (paginated).
     */
    @GET("api/photos")
    suspend fun getPhotos(
        @Query("skip") skip: Int = 0,
        @Query("take") take: Int = 50
    ): Response<PhotoListResponse>

    /**
     * Health check endpoint.
     */
    @GET("api/health")
    suspend fun healthCheck(): Response<HealthResponse>
}

// API Response/Request DTOs

data class UploadResponse(
    val id: String,
    val storedPath: String,
    val uploadedAt: String,
    val isDuplicate: Boolean
)

data class CheckHashesRequest(
    val hashes: List<String>
)

data class CheckHashesResponse(
    val existing: List<String>,
    val missing: List<String>
)

data class PhotoResponse(
    val id: String,
    val originalFilename: String,
    val storedPath: String,
    val fileSize: Long,
    val dateTaken: String,
    val uploadedAt: String
)

data class PhotoListResponse(
    val photos: List<PhotoResponse>,
    val totalCount: Int,
    val skip: Int,
    val take: Int
)

data class HealthResponse(
    val status: String,
    val timestamp: String
)

data class ErrorResponse(
    val error: String
)
