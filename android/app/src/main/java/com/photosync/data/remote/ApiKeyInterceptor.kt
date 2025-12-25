package com.photosync.data.remote

import okhttp3.Interceptor
import okhttp3.Response

/**
 * OkHttp interceptor that adds the API key header to all requests.
 */
class ApiKeyInterceptor(
    private val apiKeyProvider: () -> String
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val apiKey = apiKeyProvider()

        val request = chain.request().newBuilder()
            .apply {
                if (apiKey.isNotBlank()) {
                    addHeader(HEADER_NAME, apiKey)
                }
            }
            .build()

        return chain.proceed(request)
    }

    companion object {
        const val HEADER_NAME = "X-API-Key"
    }
}
