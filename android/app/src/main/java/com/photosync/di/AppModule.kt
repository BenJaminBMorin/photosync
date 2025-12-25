package com.photosync.di

import android.content.Context
import androidx.room.Room
import com.photosync.data.local.PhotoDatabase
import com.photosync.data.local.SyncedPhotoDao
import com.photosync.data.remote.ApiKeyInterceptor
import com.photosync.data.remote.PhotoSyncApi
import com.photosync.data.repository.SettingsRepository
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object AppModule {

    @Provides
    @Singleton
    fun providePhotoDatabase(
        @ApplicationContext context: Context
    ): PhotoDatabase {
        return Room.databaseBuilder(
            context,
            PhotoDatabase::class.java,
            PhotoDatabase.DATABASE_NAME
        ).build()
    }

    @Provides
    @Singleton
    fun provideSyncedPhotoDao(database: PhotoDatabase): SyncedPhotoDao {
        return database.syncedPhotoDao()
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(
        settingsRepository: SettingsRepository
    ): OkHttpClient {
        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        val apiKeyInterceptor = ApiKeyInterceptor {
            runBlocking { settingsRepository.getApiKey() }
        }

        return OkHttpClient.Builder()
            .addInterceptor(apiKeyInterceptor)
            .addInterceptor(loggingInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .writeTimeout(120, TimeUnit.SECONDS) // Longer timeout for uploads
            .build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(
        okHttpClient: OkHttpClient,
        settingsRepository: SettingsRepository
    ): Retrofit {
        // Default base URL - will be overridden dynamically
        val baseUrl = runBlocking {
            settingsRepository.getServerUrl().ifBlank { "http://localhost:5000" }
        }

        return Retrofit.Builder()
            .baseUrl("$baseUrl/")
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    @Provides
    @Singleton
    fun providePhotoSyncApi(retrofit: Retrofit): PhotoSyncApi {
        return retrofit.create(PhotoSyncApi::class.java)
    }
}
