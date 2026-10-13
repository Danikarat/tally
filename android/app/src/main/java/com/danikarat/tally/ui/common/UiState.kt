package com.danikarat.tally.ui.common

/**
 * UiState represents the generic state of UI screens and async operations.
 */
sealed interface UiState<out T> {
    /**
     * Initial idle state before any action is performed.
     */
    data object Idle : UiState<Nothing>

    /**
     * Async operation in progress.
     */
    data object Loading : UiState<Nothing>

    /**
     * Successful completion with data payload.
     */
    data class Success<out T>(val data: T) : UiState<T>

    /**
     * Failure state containing error message and optional cause.
     */
    data class Error(
        val message: String,
        val cause: Throwable? = null
    ) : UiState<Nothing>
}
