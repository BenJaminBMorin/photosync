package com.photosync.data.repository

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import com.photosync.domain.model.Settings
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.dataStore: DataStore<Preferences> by preferencesDataStore(name = "settings")

@Singleton
class SettingsRepository @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private object PreferenceKeys {
        val SERVER_URL = stringPreferencesKey("server_url")
        val API_KEY = stringPreferencesKey("api_key")
        val WIFI_ONLY = booleanPreferencesKey("wifi_only")
    }

    /**
     * Flow of current settings.
     */
    val settings: Flow<Settings> = context.dataStore.data.map { preferences ->
        Settings(
            serverUrl = preferences[PreferenceKeys.SERVER_URL] ?: "",
            apiKey = preferences[PreferenceKeys.API_KEY] ?: "",
            wifiOnly = preferences[PreferenceKeys.WIFI_ONLY] ?: true
        )
    }

    /**
     * Get current settings synchronously.
     */
    suspend fun getSettings(): Settings {
        return settings.first()
    }

    /**
     * Get current API key.
     */
    suspend fun getApiKey(): String {
        return context.dataStore.data.first()[PreferenceKeys.API_KEY] ?: ""
    }

    /**
     * Get current server URL.
     */
    suspend fun getServerUrl(): String {
        return context.dataStore.data.first()[PreferenceKeys.SERVER_URL] ?: ""
    }

    /**
     * Update server URL.
     */
    suspend fun setServerUrl(url: String) {
        context.dataStore.edit { preferences ->
            preferences[PreferenceKeys.SERVER_URL] = url.trimEnd('/')
        }
    }

    /**
     * Update API key.
     */
    suspend fun setApiKey(apiKey: String) {
        context.dataStore.edit { preferences ->
            preferences[PreferenceKeys.API_KEY] = apiKey
        }
    }

    /**
     * Update WiFi-only setting.
     */
    suspend fun setWifiOnly(wifiOnly: Boolean) {
        context.dataStore.edit { preferences ->
            preferences[PreferenceKeys.WIFI_ONLY] = wifiOnly
        }
    }

    /**
     * Save all settings at once.
     */
    suspend fun saveSettings(settings: Settings) {
        context.dataStore.edit { preferences ->
            preferences[PreferenceKeys.SERVER_URL] = settings.serverUrl.trimEnd('/')
            preferences[PreferenceKeys.API_KEY] = settings.apiKey
            preferences[PreferenceKeys.WIFI_ONLY] = settings.wifiOnly
        }
    }
}
