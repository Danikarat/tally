package com.danikarat.tally

import com.danikarat.tally.ui.common.ProviderAdapterResolver
import com.danikarat.tally.ui.common.ProviderVisuals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ProviderAdapterTest {

    @Test
    fun resolver_mapsMtnAliasesCorrectly() {
        assertEquals(ProviderVisuals.MTN, ProviderAdapterResolver.resolve("MTN"))
        assertEquals(ProviderVisuals.MTN, ProviderAdapterResolver.resolve("momo"))
        assertEquals(ProviderVisuals.MTN, ProviderAdapterResolver.resolve("MobileMoney"))
        assertEquals(ProviderVisuals.MTN, ProviderAdapterResolver.resolve(" mtn "))

        assertEquals("MTN MoMo", ProviderVisuals.MTN.displayName)
        assertEquals("MTN", ProviderVisuals.MTN.badgeText)
    }

    @Test
    fun resolver_mapsTelecelAliasesCorrectly() {
        assertEquals(ProviderVisuals.Telecel, ProviderAdapterResolver.resolve("TELECEL"))
        assertEquals(ProviderVisuals.Telecel, ProviderAdapterResolver.resolve("vodafone"))
        assertEquals(ProviderVisuals.Telecel, ProviderAdapterResolver.resolve("TelecelCash"))

        assertEquals("Telecel Cash", ProviderVisuals.Telecel.displayName)
        assertEquals("TELECEL", ProviderVisuals.Telecel.badgeText)
    }

    @Test
    fun resolver_mapsATMoneyAliasesCorrectly() {
        assertEquals(ProviderVisuals.ATMoney, ProviderAdapterResolver.resolve("AT"))
        assertEquals(ProviderVisuals.ATMoney, ProviderAdapterResolver.resolve("airteltigo"))
        assertEquals(ProviderVisuals.ATMoney, ProviderAdapterResolver.resolve("AtMoney"))

        assertEquals("AT Money", ProviderVisuals.ATMoney.displayName)
        assertEquals("AT", ProviderVisuals.ATMoney.badgeText)
    }

    @Test
    fun resolver_mapsUnknownAndNullToOther() {
        val nullResult = ProviderAdapterResolver.resolve(null)
        assertTrue(nullResult is ProviderVisuals.Other)
        assertEquals("MOMO", nullResult.badgeText)

        val bankResult = ProviderAdapterResolver.resolve("Ecobank")
        assertTrue(bankResult is ProviderVisuals.Other)
        assertEquals("Ecobank", (bankResult as ProviderVisuals.Other).rawName)
    }
}
