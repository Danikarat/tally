package com.danikarat.tally

import com.danikarat.tally.di.AppModule
import com.danikarat.tally.di.NetworkModule
import kotlinx.coroutines.Dispatchers
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test

class DiModulesTest {

    @Test
    fun appModule_providesValidDispatchers() {
        val io = AppModule.provideIoDispatcher()
        val default = AppModule.provideDefaultDispatcher()
        val main = AppModule.provideMainDispatcher()

        assertEquals(Dispatchers.IO, io)
        assertEquals(Dispatchers.Default, default)
        assertEquals(Dispatchers.Main, main)
    }

    @Test
    fun networkModule_providesLoggingInterceptorAndClient() {
        val interceptor = NetworkModule.provideLoggingInterceptor()
        assertNotNull(interceptor)

        val okHttpClient = NetworkModule.provideOkHttpClient(interceptor)
        assertNotNull(okHttpClient)

        val retrofit = NetworkModule.provideRetrofit(okHttpClient)
        assertNotNull(retrofit)
        assertEquals("http://10.0.2.2:8080/v1/", retrofit.baseUrl().toString())

        val apiService = NetworkModule.provideTallyApiService(retrofit)
        assertNotNull(apiService)
    }
}
