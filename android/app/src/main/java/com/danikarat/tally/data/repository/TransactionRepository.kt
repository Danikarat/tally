package com.danikarat.tally.data.repository

import com.danikarat.tally.data.local.TransactionDao
import com.danikarat.tally.data.local.entity.TransactionEntity
import com.danikarat.tally.data.remote.TallyApiService
import com.danikarat.tally.data.remote.model.TransactionDto
import kotlinx.coroutines.flow.Flow
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

/**
 * SpendingSummary aggregates monetary totals in integer minor units (pesewas).
 */
data class SpendingSummary(
    val totalInboundMinorUnits: Long = 0L,
    val totalOutboundMinorUnits: Long = 0L,
    val netMinorUnits: Long = 0L
)

/**
 * TransactionRepository coordinates local cache and remote API transactions.
 */
interface TransactionRepository {
    fun getTransactionsStream(): Flow<List<TransactionEntity>>
    suspend fun getTransactionById(id: String): TransactionEntity?
    suspend fun refreshTransactions(): Result<List<TransactionEntity>>
    fun calculateSummary(transactions: List<TransactionEntity>): SpendingSummary
}

@Singleton
class TransactionRepositoryImpl @Inject constructor(
    private val apiService: TallyApiService,
    private val transactionDao: TransactionDao
) : TransactionRepository {

    override fun getTransactionsStream(): Flow<List<TransactionEntity>> {
        return transactionDao.getAllTransactions()
    }

    override suspend fun getTransactionById(id: String): TransactionEntity? {
        return transactionDao.getTransactionById(id)
    }

    override suspend fun refreshTransactions(): Result<List<TransactionEntity>> {
        return try {
            val response = apiService.listTransactions(limit = 100, offset = 0)
            if (response.isSuccessful && response.body() != null) {
                val dtos = response.body()!!.data
                val entities = dtos.map { it.toEntity() }
                transactionDao.insertTransactions(entities)
                Result.success(entities)
            } else {
                Result.failure(Exception("Failed to fetch transactions from server: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun calculateSummary(transactions: List<TransactionEntity>): SpendingSummary {
        var inbound = 0L
        var outbound = 0L

        for (tx in transactions) {
            when (tx.flowType.uppercase().trim()) {
                "INBOUND" -> inbound += tx.amountMinorUnits
                "OUTBOUND" -> outbound += tx.amountMinorUnits
            }
        }

        return SpendingSummary(
            totalInboundMinorUnits = inbound,
            totalOutboundMinorUnits = outbound,
            netMinorUnits = inbound - outbound
        )
    }

    private fun TransactionDto.toEntity(): TransactionEntity {
        val parsedTimeMs = try {
            Instant.parse(eventTimestamp).toEpochMilli()
        } catch (e: Exception) {
            System.currentTimeMillis()
        }

        return TransactionEntity(
            id = id,
            provider = "MTN",
            amountMinorUnits = amountMinorUnits,
            currency = currency,
            flowType = direction,
            counterparty = counterparty ?: "Unknown Counterparty",
            referenceId = providerReference,
            timestampMs = parsedTimeMs,
            category = userNotes ?: "General"
        )
    }
}
