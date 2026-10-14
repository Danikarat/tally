package com.danikarat.tally

import android.content.SharedPreferences
import com.danikarat.tally.data.preferences.PreferencesManager
import com.danikarat.tally.data.preferences.PreferencesManagerImpl
import com.danikarat.tally.data.preferences.TrackingMode
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class PreferencesManagerTest {

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

    private lateinit var preferencesManager: PreferencesManager

    @Before
    fun setup() {
        preferencesManager = PreferencesManagerImpl(FakeSharedPreferences())
    }

    @Test
    fun defaultState_isCorrect() {
        assertFalse(preferencesManager.isOnboardingCompleted())
        assertEquals(TrackingMode.MANUAL, preferencesManager.getTrackingMode())
        assertFalse(preferencesManager.isSmsPermissionGranted())
    }

    @Test
    fun onboardingCompleted_persistsState() {
        preferencesManager.setOnboardingCompleted(true)
        assertTrue(preferencesManager.isOnboardingCompleted())
    }

    @Test
    fun trackingMode_persistsCorrectly() {
        preferencesManager.setTrackingMode(TrackingMode.AUTOMATIC_SMS)
        assertEquals(TrackingMode.AUTOMATIC_SMS, preferencesManager.getTrackingMode())

        preferencesManager.setTrackingMode(TrackingMode.MANUAL)
        assertEquals(TrackingMode.MANUAL, preferencesManager.getTrackingMode())
    }

    @Test
    fun smsPermission_persistsCorrectly() {
        preferencesManager.setSmsPermissionGranted(true)
        assertTrue(preferencesManager.isSmsPermissionGranted())

        preferencesManager.setSmsPermissionGranted(false)
        assertFalse(preferencesManager.isSmsPermissionGranted())
    }

    @Test
    fun clear_resetsAllPreferences() {
        preferencesManager.setOnboardingCompleted(true)
        preferencesManager.setTrackingMode(TrackingMode.AUTOMATIC_SMS)
        preferencesManager.setSmsPermissionGranted(true)

        preferencesManager.clear()

        assertFalse(preferencesManager.isOnboardingCompleted())
        assertEquals(TrackingMode.MANUAL, preferencesManager.getTrackingMode())
        assertFalse(preferencesManager.isSmsPermissionGranted())
    }
}
