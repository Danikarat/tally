package com.danikarat.tally.ui.screens.onboarding

import androidx.lifecycle.ViewModel
import com.danikarat.tally.data.preferences.PreferencesManager
import com.danikarat.tally.data.preferences.TrackingMode
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject

enum class OnboardingStep {
    DISCLOSURE,
    PERMISSION_DENIED,
    COMPLETED
}

data class OnboardingUiState(
    val step: OnboardingStep = OnboardingStep.DISCLOSURE,
    val trackingMode: TrackingMode = TrackingMode.AUTOMATIC_SMS
)

@HiltViewModel
class OnboardingViewModel @Inject constructor(
    private val preferencesManager: PreferencesManager
) : ViewModel() {

    private val _uiState = MutableStateFlow(OnboardingUiState())
    val uiState: StateFlow<OnboardingUiState> = _uiState.asStateFlow()

    fun onPermissionGranted() {
        preferencesManager.setSmsPermissionGranted(true)
        preferencesManager.setTrackingMode(TrackingMode.AUTOMATIC_SMS)
        preferencesManager.setOnboardingCompleted(true)
        _uiState.value = OnboardingUiState(
            step = OnboardingStep.COMPLETED,
            trackingMode = TrackingMode.AUTOMATIC_SMS
        )
    }

    fun onPermissionDenied() {
        preferencesManager.setSmsPermissionGranted(false)
        _uiState.value = _uiState.value.copy(step = OnboardingStep.PERMISSION_DENIED)
    }

    fun onContinueManualMode() {
        preferencesManager.setTrackingMode(TrackingMode.MANUAL)
        preferencesManager.setOnboardingCompleted(true)
        _uiState.value = OnboardingUiState(
            step = OnboardingStep.COMPLETED,
            trackingMode = TrackingMode.MANUAL
        )
    }

    fun onRetryPermission() {
        _uiState.value = _uiState.value.copy(step = OnboardingStep.DISCLOSURE)
    }
}
