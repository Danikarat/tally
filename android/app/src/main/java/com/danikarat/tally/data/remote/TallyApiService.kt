package com.danikarat.tally.data.remote

import retrofit2.Response
import retrofit2.http.GET

/**
 * TallyApiService defines the Retrofit HTTP interface for the Tally Go backend API.
 */
interface TallyApiService {
    @GET("healthz")
    suspend fun healthCheck(): Response<Map<String, String>>
}
