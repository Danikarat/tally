package com.danikarat.tally

import com.danikarat.tally.data.preferences.PreferencesManager
import com.danikarat.tally.data.preferences.TrackingMode
import com.danikarat.tally.ui.screens.onboarding.OnboardingStep
import com.danikarat.tally.ui.screens.onboarding.OnboardingViewModel
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class OnboardingViewModelTest {

    private class FakePreferencesManager : PreferencesManager {
        private var onboardingCompleted = false
        private var trackingMode = TrackingMode.MANUAL
        private var smsPermissionGranted = false

        override fun setOnboardingCompleted(completed: Boolean) {
            onboardingCompleted = completed
        }
        override fun isOnboardingCompleted(): Boolean = onboardingCompleted

        override fun setTrackingMode(mode: TrackingMode) {
            trackingMode = mode
        }
        override fun getTrackingMode(): TrackingMode = trackingMode

        override fun setSmsPermissionGranted(granted: Boolean) {
            smsPermissionGranted = granted
        }
        override fun isSmsPermissionGranted(): Boolean = smsPermissionGranted

        override fun clear() {
            onboardingCompleted = false
            trackingMode = TrackingMode.MANUAL
            smsPermissionGranted = false
        }
    }

    private lateinit var preferencesManager: FakePreferencesManager
    private lateinit var viewModel: OnboardingViewModel

    @Before
    fun setup() {
        preferencesManager = FakePreferencesManager()
        viewModel = OnboardingViewModel(preferencesManager)
    }

    @Test
    fun initialState_isDisclosure() {
        assertEquals(OnboardingStep.DISCLOSURE, viewModel.uiState.value.step)
    }

    @Test
    fun onPermissionGranted_enablesAutomaticTrackingAndCompletes() {
        viewModel.onPermissionGranted()

        assertEquals(OnboardingStep.COMPLETED, viewModel.uiState.value.step)
        assertEquals(TrackingMode.AUTOMATIC_SMS, viewModel.uiState.value.trackingMode)
        assertTrue(preferencesManager.isSmsPermissionGranted())
        assertEquals(TrackingMode.AUTOMATIC_SMS, preferencesManager.getTrackingMode())
        assertTrue(preferencesManager.isOnboardingCompleted())
    }

    @Test
    fun onPermissionDenied_transitionsToDeniedState() {
        viewModel.onPermissionDenied()

        assertEquals(OnboardingStep.PERMISSION_DENIED, viewModel.uiState.value.step)
        assertFalse(preferencesManager.isSmsPermissionGranted())
        assertFalse(preferencesManager.isOnboardingCompleted())
    }

    @Test
    fun onContinueManualMode_enablesManualTrackingAndCompletes() {
        viewModel.onPermissionDenied()
        assertEquals(OnboardingStep.PERMISSION_DENIED, viewModel.uiState.value.step)

        viewModel.onContinueManualMode()

        assertEquals(OnboardingStep.COMPLETED, viewModel.uiState.value.step)
        assertEquals(TrackingMode.MANUAL, viewModel.uiState.value.trackingMode)
        assertFalse(preferencesManager.isSmsPermissionGranted())
        assertEquals(TrackingMode.MANUAL, preferencesManager.getTrackingMode())
        assertTrue(preferencesManager.isOnboardingCompleted())
    }

    @Test
    fun onRetryPermission_returnsToDisclosureStep() {
        viewModel.onPermissionDenied()
        assertEquals(OnboardingStep.PERMISSION_DENIED, viewModel.uiState.value.step)

        viewModel.onRetryPermission()
        assertEquals(OnboardingStep.DISCLOSURE, viewModel.uiState.value.step)
    }
}
