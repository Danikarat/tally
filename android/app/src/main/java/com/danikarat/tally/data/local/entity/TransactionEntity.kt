package com.danikarat.tally.data.local.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * TransactionEntity stores local cached transactions.
 * All monetary amounts are stored as integer minor units (pesewas) to prevent floating-point inaccuracy.
 */
@Entity(tableName = "transactions")
data class TransactionEntity(
    @PrimaryKey
    val id: String,
    val provider: String,
    val amountMinorUnits: Long,
    val currency: String = "GHS",
    val flowType: String,
    val counterparty: String,
    val referenceId: String?,
    val timestampMs: Long,
    val category: String = "Uncategorized"
)
