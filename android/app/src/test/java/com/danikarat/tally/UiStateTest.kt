package com.danikarat.tally

import com.danikarat.tally.ui.common.UiState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class UiStateTest {

    @Test
    fun idleState_isCorrect() {
        val state: UiState<String> = UiState.Idle
        assertTrue(state is UiState.Idle)
    }

    @Test
    fun loadingState_isCorrect() {
        val state: UiState<String> = UiState.Loading
        assertTrue(state is UiState.Loading)
    }

    @Test
    fun successState_holdsExpectedData() {
        val expected = "ghana_cedi_balance"
        val state: UiState<String> = UiState.Success(expected)

        assertTrue(state is UiState.Success)
        assertEquals(expected, (state as UiState.Success).data)
    }

    @Test
    fun errorState_holdsMessageAndCause() {
        val cause = IllegalStateException("Network unreachable")
        val state: UiState<String> = UiState.Error("Failed to fetch transactions", cause)

        assertTrue(state is UiState.Error)
        val error = state as UiState.Error
        assertEquals("Failed to fetch transactions", error.message)
        assertEquals(cause, error.cause)
    }

    @Test
    fun errorState_withoutCause_defaultsToNull() {
        val state: UiState<String> = UiState.Error("Unknown error")

        assertTrue(state is UiState.Error)
        val error = state as UiState.Error
        assertEquals("Unknown error", error.message)
        assertNull(error.cause)
    }
}
