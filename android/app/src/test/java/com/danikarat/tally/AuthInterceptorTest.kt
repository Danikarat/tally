package com.danikarat.tally

import com.danikarat.tally.data.remote.AuthInterceptor
import com.danikarat.tally.data.security.TokenManager
import okhttp3.Interceptor
import okhttp3.Protocol
import okhttp3.Request
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.concurrent.atomic.AtomicBoolean

class AuthInterceptorTest {

    private class FakeTokenManager(var token: String? = null) : TokenManager {
        var clearCalled = false
        override fun saveToken(token: String) { this.token = token }
        override fun getToken(): String? = token
        override fun clearToken() {
            this.token = null
            clearCalled = true
        }
    }

    private class FakeChain(
        private val request: Request,
        private val responseCode: Int = 200
    ) : Interceptor.Chain {
        var capturedRequest: Request? = null

        override fun request(): Request = request
        override fun proceed(request: Request): Response {
            capturedRequest = request
            return Response.Builder()
                .request(request)
                .protocol(Protocol.HTTP_1_1)
                .code(responseCode)
                .message(if (responseCode == 200) "OK" else "Unauthorized")
                .body("{}".toResponseBody())
                .build()
        }

        override fun connection(): okhttp3.Connection? = null
        override fun call(): okhttp3.Call = throw UnsupportedOperationException()
        override fun connectTimeoutMillis(): Int = 0
        override fun withConnectTimeout(timeout: Int, unit: java.util.concurrent.TimeUnit): Interceptor.Chain = this
        override fun readTimeoutMillis(): Int = 0
        override fun withReadTimeout(timeout: Int, unit: java.util.concurrent.TimeUnit): Interceptor.Chain = this
        override fun writeTimeoutMillis(): Int = 0
        override fun withWriteTimeout(timeout: Int, unit: java.util.concurrent.TimeUnit): Interceptor.Chain = this
    }

    @Test
    fun interceptor_attachesBearerToken_whenTokenPresent() {
        val fakeTokenManager = FakeTokenManager(token = "valid_jwt_token")
        val interceptor = AuthInterceptor(fakeTokenManager)

        val originalRequest = Request.Builder()
            .url("https://api.tally.money/v1/transactions")
            .build()
        val fakeChain = FakeChain(originalRequest, responseCode = 200)

        interceptor.intercept(fakeChain)

        val authorizationHeader = fakeChain.capturedRequest?.header("Authorization")
        assertEquals("Bearer valid_jwt_token", authorizationHeader)
    }

    @Test
    fun interceptor_doesNotAttachHeader_whenTokenNull() {
        val fakeTokenManager = FakeTokenManager(token = null)
        val interceptor = AuthInterceptor(fakeTokenManager)

        val originalRequest = Request.Builder()
            .url("https://api.tally.money/v1/auth/login")
            .build()
        val fakeChain = FakeChain(originalRequest, responseCode = 200)

        interceptor.intercept(fakeChain)

        val authorizationHeader = fakeChain.capturedRequest?.header("Authorization")
        assertNull(authorizationHeader)
    }

    @Test
    fun interceptor_clearsTokenAndInvokesCallback_on401Unauthorized() {
        val fakeTokenManager = FakeTokenManager(token = "expired_jwt_token")
        val unauthorizedInvoked = AtomicBoolean(false)
        val interceptor = AuthInterceptor(fakeTokenManager, onUnauthorized = {
            unauthorizedInvoked.set(true)
        })

        val originalRequest = Request.Builder()
            .url("https://api.tally.money/v1/transactions")
            .build()
        val fakeChain = FakeChain(originalRequest, responseCode = 401)

        val response = interceptor.intercept(fakeChain)

        assertEquals(401, response.code)
        assertTrue(fakeTokenManager.clearCalled)
        assertNull(fakeTokenManager.getToken())
        assertTrue(unauthorizedInvoked.get())
    }
}
