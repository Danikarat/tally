package com.danikarat.tally.data.security

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * TokenManager defines secure storage operations for authentication tokens.
 */
interface TokenManager {
    fun saveToken(token: String)
    fun getToken(): String?
    fun clearToken()
    fun hasToken(): Boolean = !getToken().isNullOrBlank()
}

/**
 * EncryptedTokenManager implements TokenManager using Android Keystore-backed EncryptedSharedPreferences.
 */
class EncryptedTokenManager(
    private val preferences: SharedPreferences
) : TokenManager {

    companion object {
        private const val PREFS_FILE_NAME = "tally_secure_prefs"
        private const val KEY_AUTH_TOKEN = "key_jwt_auth_token"

        fun create(context: Context): EncryptedTokenManager {
            val masterKey = MasterKey.Builder(context)
                .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                .build()

            val prefs = EncryptedSharedPreferences.create(
                context,
                PREFS_FILE_NAME,
                masterKey,
                EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
            )
            return EncryptedTokenManager(prefs)
        }
    }

    override fun saveToken(token: String) {
        preferences.edit().putString(KEY_AUTH_TOKEN, token).apply()
    }

    override fun getToken(): String? {
        return preferences.getString(KEY_AUTH_TOKEN, null)
    }

    override fun clearToken() {
        preferences.edit().remove(KEY_AUTH_TOKEN).apply()
    }
}
