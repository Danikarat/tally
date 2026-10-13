package com.danikarat.tally.ui.common

import java.util.Locale
import kotlin.math.abs

/**
 * CurrencyFormatter formats minor units (pesewas) to Ghanaian Cedis (GH₵)
 * using deterministic integer/Long arithmetic to eliminate floating-point rounding errors.
 */
object CurrencyFormatter {

    const val DEFAULT_CURRENCY_SYMBOL = "GH₵"

    /**
     * Formats an amount in minor units (pesewas) into standard Ghanaian Cedi representation.
     * e.g. 5000L -> "GH₵ 50.00", 75L -> "GH₵ 0.75", -12550L -> "-GH₵ 125.50"
     */
    fun formatPesewas(
        amountMinorUnits: Long,
        currencySymbol: String = DEFAULT_CURRENCY_SYMBOL
    ): String {
        val isNegative = amountMinorUnits < 0
        val absUnits = abs(amountMinorUnits)
        val cedis = absUnits / 100
        val pesewas = absUnits % 100

        val formattedPesewas = String.format(Locale.US, "%02d", pesewas)
        val result = "$currencySymbol $cedis.$formattedPesewas"

        return if (isNegative) "-$result" else result
    }

    /**
     * Formats an amount with an explicit +/- sign based on transaction direction.
     * INBOUND -> "+GH₵ 50.00"
     * OUTBOUND -> "-GH₵ 50.00"
     */
    fun formatTransactionAmount(
        amountMinorUnits: Long,
        direction: String,
        currencySymbol: String = DEFAULT_CURRENCY_SYMBOL
    ): String {
        val absUnits = abs(amountMinorUnits)
        val formatted = formatPesewas(absUnits, currencySymbol)

        return when (direction.uppercase().trim()) {
            "INBOUND" -> "+$formatted"
            "OUTBOUND" -> "-$formatted"
            else -> formatted
        }
    }
}
