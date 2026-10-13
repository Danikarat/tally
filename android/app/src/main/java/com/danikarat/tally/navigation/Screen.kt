package com.danikarat.tally.navigation

/**
 * Screen defines the navigation routes for Tally.
 */
sealed class Screen(val route: String) {
    data object Auth : Screen("auth")
    data object Onboarding : Screen("onboarding")
    data object Dashboard : Screen("dashboard")
    data object TransactionDetail : Screen("transaction_detail/{transactionId}") {
        fun createRoute(transactionId: String): String = "transaction_detail/$transactionId"
        const val ARG_TRANSACTION_ID = "transactionId"
    }
    data object Settings : Screen("settings")
}
