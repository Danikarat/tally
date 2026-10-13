package com.danikarat.tally

import com.danikarat.tally.ui.common.CurrencyFormatter
import org.junit.Assert.assertEquals
import org.junit.Test

class CurrencyFormatterTest {

    @Test
    fun formatPesewas_zero_returnsCorrectString() {
        assertEquals("GH₵ 0.00", CurrencyFormatter.formatPesewas(0L))
    }

    @Test
    fun formatPesewas_singleDigitPesewas() {
        assertEquals("GH₵ 0.05", CurrencyFormatter.formatPesewas(5L))
        assertEquals("GH₵ 0.01", CurrencyFormatter.formatPesewas(1L))
    }

    @Test
    fun formatPesewas_doubleDigitPesewas() {
        assertEquals("GH₵ 0.75", CurrencyFormatter.formatPesewas(75L))
        assertEquals("GH₵ 0.99", CurrencyFormatter.formatPesewas(99L))
    }

    @Test
    fun formatPesewas_standardCedisAndPesewas() {
        assertEquals("GH₵ 50.00", CurrencyFormatter.formatPesewas(5000L))
        assertEquals("GH₵ 125.50", CurrencyFormatter.formatPesewas(12550L))
        assertEquals("GH₵ 1000.00", CurrencyFormatter.formatPesewas(100000L))
    }

    @Test
    fun formatPesewas_negativeAmounts() {
        assertEquals("-GH₵ 50.00", CurrencyFormatter.formatPesewas(-5000L))
        assertEquals("-GH₵ 0.50", CurrencyFormatter.formatPesewas(-50L))
    }

    @Test
    fun formatTransactionAmount_inboundDirection_prefixesPlus() {
        assertEquals("+GH₵ 50.00", CurrencyFormatter.formatTransactionAmount(5000L, "INBOUND"))
        assertEquals("+GH₵ 50.00", CurrencyFormatter.formatTransactionAmount(5000L, "inbound"))
    }

    @Test
    fun formatTransactionAmount_outboundDirection_prefixesMinus() {
        assertEquals("-GH₵ 125.00", CurrencyFormatter.formatTransactionAmount(12500L, "OUTBOUND"))
        assertEquals("-GH₵ 125.00", CurrencyFormatter.formatTransactionAmount(12500L, "outbound"))
    }

    @Test
    fun formatTransactionAmount_unknownDirection_omitsSign() {
        assertEquals("GH₵ 10.00", CurrencyFormatter.formatTransactionAmount(1000L, "UNKNOWN"))
    }
}
