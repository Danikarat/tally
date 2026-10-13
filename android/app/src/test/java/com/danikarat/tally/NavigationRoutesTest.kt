package com.danikarat.tally

import com.danikarat.tally.navigation.Screen
import org.junit.Assert.assertEquals
import org.junit.Test

class NavigationRoutesTest {

    @Test
    fun authRoute_isCorrect() {
        assertEquals("auth", Screen.Auth.route)
    }

    @Test
    fun onboardingRoute_isCorrect() {
        assertEquals("onboarding", Screen.Onboarding.route)
    }

    @Test
    fun dashboardRoute_isCorrect() {
        assertEquals("dashboard", Screen.Dashboard.route)
    }

    @Test
    fun transactionDetailRoute_patternIsCorrect() {
        assertEquals("transaction_detail/{transactionId}", Screen.TransactionDetail.route)
    }

    @Test
    fun transactionDetail_createRoute_formatsProperly() {
        val txId = "tx_momo_gh_98765"
        val expected = "transaction_detail/tx_momo_gh_98765"
        assertEquals(expected, Screen.TransactionDetail.createRoute(txId))
    }

    @Test
    fun settingsRoute_isCorrect() {
        assertEquals("settings", Screen.Settings.route)
    }
}
