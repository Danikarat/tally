package com.danikarat.tally

import android.content.SharedPreferences
import com.danikarat.tally.data.security.EncryptedTokenManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class TokenManagerTest {

    private class FakeSharedPreferences : SharedPreferences {
        private val storage = mutableMapOf<String, Any?>()

        override fun getAll(): MutableMap<String, *> = storage
        override fun getString(key: String?, defValue: String?): String? = storage[key] as? String ?: defValue
        override fun getStringSet(key: String?, defValues: MutableSet<String>?): MutableSet<String>? = null
        override fun getInt(key: String?, defValue: Int): Int = (storage[key] as? Int) ?: defValue
        override fun getLong(key: String?, defValue: Long): Long = (storage[key] as? Long) ?: defValue
        override fun getFloat(key: String?, defValue: Float): Float = (storage[key] as? Float) ?: defValue
        override fun getBoolean(key: String?, defValue: Boolean): Boolean = (storage[key] as? Boolean) ?: defValue
        override fun contains(key: String?): Boolean = storage.containsKey(key)
        override fun registerOnSharedPreferenceChangeListener(listener: SharedPreferences.OnSharedPreferenceChangeListener?) {}
        override fun unregisterOnSharedPreferenceChangeListener(listener: SharedPreferences.OnSharedPreferenceChangeListener?) {}

        override fun edit(): SharedPreferences.Editor = FakeEditor(storage)

        private class FakeEditor(private val storage: MutableMap<String, Any?>) : SharedPreferences.Editor {
            private val pending = mutableMapOf<String, Any?>()
            private val removes = mutableSetOf<String>()

            override fun putString(key: String?, value: String?): SharedPreferences.Editor {
                if (key != null) pending[key] = value
                return this
            }
            override fun putStringSet(key: String?, values: MutableSet<String>?): SharedPreferences.Editor = this
            override fun putInt(key: String?, value: Int): SharedPreferences.Editor = this
            override fun putLong(key: String?, value: Long): SharedPreferences.Editor = this
            override fun putFloat(key: String?, value: Float): SharedPreferences.Editor = this
            override fun putBoolean(key: String?, value: Boolean): SharedPreferences.Editor = this
            override fun remove(key: String?): SharedPreferences.Editor {
                if (key != null) removes.add(key)
                return this
            }
            override fun clear(): SharedPreferences.Editor {
                storage.clear()
                return this
            }
            override fun commit(): Boolean {
                apply()
                return true
            }
            override fun apply() {
                removes.forEach { storage.remove(it) }
                pending.forEach { (k, v) -> storage[k] = v }
            }
        }
    }

    private lateinit var tokenManager: EncryptedTokenManager

    @Before
    fun setup() {
        tokenManager = EncryptedTokenManager(FakeSharedPreferences())
    }

    @Test
    fun initialToken_isNull() {
        assertNull(tokenManager.getToken())
        assertFalse(tokenManager.hasToken())
    }

    @Test
    fun saveToken_persistsToken() {
        val sampleToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_token"
        tokenManager.saveToken(sampleToken)

        assertEquals(sampleToken, tokenManager.getToken())
        assertTrue(tokenManager.hasToken())
    }

    @Test
    fun clearToken_removesToken() {
        tokenManager.saveToken("sample_token")
        assertTrue(tokenManager.hasToken())

        tokenManager.clearToken()
        assertNull(tokenManager.getToken())
        assertFalse(tokenManager.hasToken())
    }
}
