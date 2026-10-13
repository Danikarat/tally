package com.danikarat.tally.data.remote.model

import com.google.gson.annotations.SerializedName

/**
 * RegisterRequest holds inputs for user registration.
 */
data class RegisterRequest(
    @SerializedName("email") val email: String,
    @SerializedName("password") val password: String,
    @SerializedName("full_name") val fullName: String
)

/**
 * LoginRequest holds credentials for user login.
 */
data class LoginRequest(
    @SerializedName("email") val email: String,
    @SerializedName("password") val password: String
)

/**
 * UserDto represents the public user data model returned by the Go backend.
 */
data class UserDto(
    @SerializedName("id") val id: String,
    @SerializedName("email") val email: String,
    @SerializedName("full_name") val fullName: String,
    @SerializedName("is_active") val isActive: Boolean,
    @SerializedName("created_at") val createdAt: String
)

/**
 * AuthResponse holds the authentication token and user profile returned by the backend.
 */
data class AuthResponse(
    @SerializedName("user") val user: UserDto,
    @SerializedName("token") val token: String,
    @SerializedName("expires_at") val expiresAt: String
)

/**
 * UserResponse represents response from /v1/auth/me.
 */
data class UserResponse(
    @SerializedName("user") val user: UserDto
)

/**
 * ErrorResponse represents backend JSON error response.
 */
data class ErrorResponse(
    @SerializedName("error") val error: String
)
