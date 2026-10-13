package com.danikarat.tally.data.remote

import com.danikarat.tally.data.remote.model.AuthResponse
import com.danikarat.tally.data.remote.model.LoginRequest
import com.danikarat.tally.data.remote.model.RegisterRequest
import com.danikarat.tally.data.remote.model.UserResponse
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

/**
 * TallyApiService defines the Retrofit HTTP interface for the Tally Go backend API.
 */
interface TallyApiService {
    @GET("healthz")
    suspend fun healthCheck(): Response<Map<String, String>>

    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthResponse>

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthResponse>

    @GET("auth/me")
    suspend fun getMe(): Response<UserResponse>
}
