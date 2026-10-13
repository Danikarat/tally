package com.danikarat.tally

import com.danikarat.tally.data.remote.TallyApiService
import com.danikarat.tally.data.remote.model.AuthResponse
import com.danikarat.tally.data.remote.model.LoginRequest
import com.danikarat.tally.data.remote.model.RegisterRequest
import com.danikarat.tally.data.remote.model.UserDto
import com.danikarat.tally.data.remote.model.UserResponse
import com.danikarat.tally.data.repository.AuthRepositoryImpl
import com.danikarat.tally.data.security.TokenManager
import kotlinx.coroutines.test.runTest
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Response

class AuthRepositoryTest {

    private class FakeTokenManager : TokenManager {
        private var token: String? = null
        override fun saveToken(token: String) { this.token = token }
        override fun getToken(): String? = token
        override fun clearToken() { this.token = null }
    }

    private class FakeApiService : TallyApiService {
        var registerResponse: Response<AuthResponse>? = null
        var loginResponse: Response<AuthResponse>? = null
        var getMeResponse: Response<UserResponse>? = null

        override suspend fun healthCheck(): Response<Map<String, String>> = Response.success(mapOf("status" to "ok"))
        override suspend fun register(request: RegisterRequest): Response<AuthResponse> = registerResponse ?: throw IllegalStateException()
        override suspend fun login(request: LoginRequest): Response<AuthResponse> = loginResponse ?: throw IllegalStateException()
        override suspend fun getMe(): Response<UserResponse> = getMeResponse ?: throw IllegalStateException()
    }

    private lateinit var tokenManager: FakeTokenManager
    private lateinit var apiService: FakeApiService
    private lateinit var repository: AuthRepositoryImpl

    @Before
    fun setup() {
        tokenManager = FakeTokenManager()
        apiService = FakeApiService()
        repository = AuthRepositoryImpl(apiService, tokenManager)
    }

    @Test
    fun login_success_savesTokenAndReturnsResponse() = runTest {
        val userDto = UserDto(
            id = "user_123",
            email = "kwame.mensah@example.com",
            fullName = "Kwame Mensah",
            isActive = true,
            createdAt = "2026-10-13T10:00:00Z"
        )
        val expectedToken = "jwt_access_token_123"
        val authResponse = AuthResponse(user = userDto, token = expectedToken, expiresAt = "2026-10-14T10:00:00Z")

        apiService.loginResponse = Response.success(authResponse)

        val result = repository.login("kwame.mensah@example.com", "StrongP@ssw0rd1")

        assertTrue(result.isSuccess)
        assertEquals(expectedToken, tokenManager.getToken())
        assertTrue(repository.isLoggedIn())
    }

    @Test
    fun login_failure_returnsErrorAndLeavesTokenEmpty() = runTest {
        val errorBody = "{\"error\":\"invalid email or password\"}".toResponseBody("application/json".toMediaType())
        apiService.loginResponse = Response.error(401, errorBody)

        val result = repository.login("kwame.mensah@example.com", "WrongPassword1")

        assertTrue(result.isFailure)
        assertEquals("invalid email or password", result.exceptionOrNull()?.message)
        assertFalse(repository.isLoggedIn())
    }

    @Test
    fun logout_clearsToken() {
        tokenManager.saveToken("sample_token")
        assertTrue(repository.isLoggedIn())

        repository.logout()
        assertFalse(repository.isLoggedIn())
    }
}
