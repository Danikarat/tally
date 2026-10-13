package com.danikarat.tally.data.remote

import com.danikarat.tally.data.security.TokenManager
import okhttp3.Interceptor
import okhttp3.Response

/**
 * AuthInterceptor attaches Bearer JWT authorization headers to outgoing HTTP requests
 * and clears stored tokens when an HTTP 401 Unauthorized status is received.
 */
class AuthInterceptor(
    private val tokenManager: TokenManager,
    private val onUnauthorized: (() -> Unit)? = null
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val originalRequest = chain.request()
        val requestBuilder = originalRequest.newBuilder()

        val token = tokenManager.getToken()
        if (!token.isNullOrBlank() && originalRequest.header("Authorization") == null) {
            requestBuilder.header("Authorization", "Bearer $token")
        }

        val response = chain.proceed(requestBuilder.build())

        if (response.code == 401) {
            tokenManager.clearToken()
            onUnauthorized?.invoke()
        }

        return response
    }
}
