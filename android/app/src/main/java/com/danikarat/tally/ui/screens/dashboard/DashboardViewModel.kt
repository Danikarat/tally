package com.danikarat.tally.ui.screens.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.danikarat.tally.data.local.entity.TransactionEntity
import com.danikarat.tally.data.repository.SpendingSummary
import com.danikarat.tally.data.repository.TransactionRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * DashboardUiState encapsulates transactions, monthly spend summary, and sync status.
 */
data class DashboardUiState(
    val transactions: List<TransactionEntity> = emptyList(),
    val summary: SpendingSummary = SpendingSummary(),
    val isRefreshing: Boolean = false,
    val errorMessage: String? = null
)

@HiltViewModel
class DashboardViewModel @Inject constructor(
    private val transactionRepository: TransactionRepository
) : ViewModel() {

    private val _isRefreshing = MutableStateFlow(false)
    private val _errorMessage = MutableStateFlow<String?>(null)

    val uiState: StateFlow<DashboardUiState> = combine(
        transactionRepository.getTransactionsStream(),
        _isRefreshing,
        _errorMessage
    ) { transactions, isRefreshing, error ->
        DashboardUiState(
            transactions = transactions,
            summary = transactionRepository.calculateSummary(transactions),
            isRefreshing = isRefreshing,
            errorMessage = error
        )
    }.stateIn(
        scope = viewModelScope,
        started = SharingStarted.WhileSubscribed(5000),
        initialValue = DashboardUiState()
    )

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _isRefreshing.value = true
            _errorMessage.value = null
            val result = transactionRepository.refreshTransactions()
            if (result.isFailure) {
                _errorMessage.value = result.exceptionOrNull()?.message ?: "Failed to refresh"
            }
            _isRefreshing.value = false
        }
    }

    fun clearError() {
        _errorMessage.value = null
    }
}
