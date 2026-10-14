package com.danikarat.tally.data.preferences

import android.content.Context
import android.content.SharedPreferences

enum class TrackingMode {
    AUTOMATIC_SMS,
    MANUAL
}

/**
 * PreferencesManager handles user onboarding and tracking mode preferences.
 */
interface PreferencesManager {
    fun setOnboardingCompleted(completed: Boolean)
    fun isOnboardingCompleted(): Boolean
    fun setTrackingMode(mode: TrackingMode)
    fun getTrackingMode(): TrackingMode
    fun setSmsPermissionGranted(granted: Boolean)
    fun isSmsPermissionGranted(): Boolean
    fun clear()
}

class PreferencesManagerImpl(
    private val preferences: SharedPreferences
) : PreferencesManager {

    companion object {
        private const val PREFS_NAME = "tally_app_preferences"
        private const val KEY_ONBOARDING_COMPLETED = "key_onboarding_completed"
        private const val KEY_TRACKING_MODE = "key_tracking_mode"
        private const val KEY_SMS_PERMISSION_GRANTED = "key_sms_permission_granted"

        fun create(context: Context): PreferencesManager {
            val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            return PreferencesManagerImpl(prefs)
        }
    }

    override fun setOnboardingCompleted(completed: Boolean) {
        preferences.edit().putBoolean(KEY_ONBOARDING_COMPLETED, completed).apply()
    }

    override fun isOnboardingCompleted(): Boolean {
        return preferences.getBoolean(KEY_ONBOARDING_COMPLETED, false)
    }

    override fun setTrackingMode(mode: TrackingMode) {
        preferences.edit().putString(KEY_TRACKING_MODE, mode.name).apply()
    }

    override fun getTrackingMode(): TrackingMode {
        val name = preferences.getString(KEY_TRACKING_MODE, TrackingMode.MANUAL.name)
        return try {
            TrackingMode.valueOf(name ?: TrackingMode.MANUAL.name)
        } catch (e: Exception) {
            TrackingMode.MANUAL
        }
    }

    override fun setSmsPermissionGranted(granted: Boolean) {
        preferences.edit().putBoolean(KEY_SMS_PERMISSION_GRANTED, granted).apply()
    }

    override fun isSmsPermissionGranted(): Boolean {
        return preferences.getBoolean(KEY_SMS_PERMISSION_GRANTED, false)
    }

    override fun clear() {
        preferences.edit().clear().apply()
    }
}
