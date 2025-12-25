package com.photosync.domain.model

/**
 * User settings for the app.
 */
data class Settings(
    val serverUrl: String = "",
    val apiKey: String = "",
    val wifiOnly: Boolean = true
) {
    val isConfigured: Boolean
        get() = serverUrl.isNotBlank() && apiKey.isNotBlank()
}
