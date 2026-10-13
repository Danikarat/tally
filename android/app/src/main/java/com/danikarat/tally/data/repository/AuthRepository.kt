package com.danikarat.tally.data.repository

import com.danikarat.tally.data.remote.TallyApiService
import com.danikarat.tally.data.remote.model.AuthResponse
import com.danikarat.tally.data.remote.model.ErrorResponse
import com.danikarat.tally.data.remote.model.LoginRequest
import com.danikarat.tally.data.remote.model.RegisterRequest
import com.danikarat.tally.data.remote.model.UserDto
import com.danikarat.tally.data.security.TokenManager
import com.google.gson.Gson
import javax.inject.Inject
import javax.inject.Singleton

/**
 * AuthRepository coordinates user authentication and credential persistence.
 */
interface AuthRepository {
    suspend fun register(email: String, password: String, fullName: String): Result<AuthResponse>
    suspend fun login(email: String, password: String): Result<AuthResponse>
    suspend fun getMe(): Result<UserDto>
    fun logout()
    fun isLoggedIn(): Boolean
}

@Singleton
class AuthRepositoryImpl @Inject constructor(
    private val apiService: TallyApiService,
    private val tokenManager: TokenManager
) : AuthRepository {

    private val gson = Gson()

    override suspend fun register(email: String, password: String, fullName: String): Result<AuthResponse> {
        return try {
            val response = apiService.register(RegisterRequest(email = email.trim(), password = password, fullName = fullName.trim()))
            if (response.isSuccessful && response.body() != null) {
                val authResponse = response.body()!!
                tokenManager.saveToken(authResponse.token)
                Result.success(authResponse)
            } else {
                val errorMessage = parseErrorMessage(response.errorBody()?.string())
                Result.failure(Exception(errorMessage))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun login(email: String, password: String): Result<AuthResponse> {
        return try {
            val response = apiService.login(LoginRequest(email = email.trim(), password = password))
            if (response.isSuccessful && response.body() != null) {
                val authResponse = response.body()!!
                tokenManager.saveToken(authResponse.token)
                Result.success(authResponse)
            } else {
                val errorMessage = parseErrorMessage(response.errorBody()?.string())
                Result.failure(Exception(errorMessage))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getMe(): Result<UserDto> {
        return try {
            val response = apiService.getMe()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!.user)
            } else {
                val errorMessage = parseErrorMessage(response.errorBody()?.string())
                Result.failure(Exception(errorMessage))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun logout() {
        tokenManager.clearToken()
    }

    override fun isLoggedIn(): Boolean {
        return tokenManager.hasToken()
    }

    private fun parseErrorMessage(errorBody: String?): String {
        if (errorBody.isNullOrBlank()) return "Unknown error occurred"
        return try {
            val errorResponse = gson.fromJson(errorBody, ErrorResponse::class.java)
            errorResponse.error.ifBlank { "Request failed" }
        } catch (e: Exception) {
            errorBody
        }
    }
}
