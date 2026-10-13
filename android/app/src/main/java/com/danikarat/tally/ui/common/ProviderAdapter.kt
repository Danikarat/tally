package com.danikarat.tally.ui.common

import androidx.compose.ui.graphics.Color

/**
 * ProviderVisuals defines the presentation metadata for Ghanaian payment providers.
 */
sealed interface ProviderVisuals {
    val providerId: String
    val displayName: String
    val badgeText: String
    val badgeBackgroundColor: Color
    val badgeTextColor: Color

    data object MTN : ProviderVisuals {
        override val providerId: String = "MTN"
        override val displayName: String = "MTN MoMo"
        override val badgeText: String = "MTN"
        override val badgeBackgroundColor: Color = Color(0xFFFFCC00)
        override val badgeTextColor: Color = Color(0xFF000000)
    }

    data object Telecel : ProviderVisuals {
        override val providerId: String = "TELECEL"
        override val displayName: String = "Telecel Cash"
        override val badgeText: String = "TELECEL"
        override val badgeBackgroundColor: Color = Color(0xFFE60000)
        override val badgeTextColor: Color = Color(0xFFFFFFFF)
    }

    data object ATMoney : ProviderVisuals {
        override val providerId: String = "AT"
        override val displayName: String = "AT Money"
        override val badgeText: String = "AT"
        override val badgeBackgroundColor: Color = Color(0xFF0055A5)
        override val badgeTextColor: Color = Color(0xFFFFFFFF)
    }

    data class Other(val rawName: String) : ProviderVisuals {
        override val providerId: String = "OTHER"
        override val displayName: String = rawName.ifBlank { "Mobile Money" }
        override val badgeText: String = "MOMO"
        override val badgeBackgroundColor: Color = Color(0xFF00897B)
        override val badgeTextColor: Color = Color(0xFFFFFFFF)
    }
}

/**
 * ProviderAdapterResolver dispatches provider strings to their visual adapters via a switch.
 */
object ProviderAdapterResolver {
    fun resolve(rawProvider: String?): ProviderVisuals {
        if (rawProvider == null) return ProviderVisuals.Other("")
        return when (rawProvider.uppercase().trim()) {
            "MTN", "MOMO", "MOBILEMONEY" -> ProviderVisuals.MTN
            "TELECEL", "VODAFONE", "TELECELCASH" -> ProviderVisuals.Telecel
            "AT", "AIRTELTIGO", "ATMONEY" -> ProviderVisuals.ATMoney
            else -> ProviderVisuals.Other(rawProvider)
        }
    }
}
