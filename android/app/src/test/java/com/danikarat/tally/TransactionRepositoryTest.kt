package com.danikarat.tally

import com.danikarat.tally.data.local.TransactionDao
import com.danikarat.tally.data.local.entity.TransactionEntity
import com.danikarat.tally.data.remote.TallyApiService
import com.danikarat.tally.data.remote.model.AuthResponse
import com.danikarat.tally.data.remote.model.ListTransactionsResponse
import com.danikarat.tally.data.remote.model.LoginRequest
import com.danikarat.tally.data.remote.model.PaginationMetaDto
import com.danikarat.tally.data.remote.model.RegisterRequest
import com.danikarat.tally.data.remote.model.SingleTransactionResponse
import com.danikarat.tally.data.remote.model.TransactionDto
import com.danikarat.tally.data.remote.model.UserResponse
import com.danikarat.tally.data.repository.TransactionRepositoryImpl
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Response

class TransactionRepositoryTest {

    private class FakeTransactionDao : TransactionDao {
        private val list = mutableListOf<TransactionEntity>()
        private val flow = MutableStateFlow<List<TransactionEntity>>(emptyList())

        override fun getAllTransactions(): Flow<List<TransactionEntity>> = flow

        override suspend fun getTransactionById(id: String): TransactionEntity? {
            return list.find { it.id == id }
        }

        override suspend fun insertTransactions(transactions: List<TransactionEntity>) {
            list.removeAll { existing -> transactions.any { it.id == existing.id } }
            list.addAll(transactions)
            flow.value = list.toList()
        }

        override suspend fun clearAll() {
            list.clear()
            flow.value = emptyList()
        }
    }

    private class FakeApiService : TallyApiService {
        var listResponse: Response<ListTransactionsResponse>? = null

        override suspend fun healthCheck(): Response<Map<String, String>> = Response.success(emptyMap())
        override suspend fun register(request: RegisterRequest): Response<AuthResponse> = throw UnsupportedOperationException()
        override suspend fun login(request: LoginRequest): Response<AuthResponse> = throw UnsupportedOperationException()
        override suspend fun getMe(): Response<UserResponse> = throw UnsupportedOperationException()

        override suspend fun listTransactions(limit: Int, offset: Int): Response<ListTransactionsResponse> {
            return listResponse ?: throw IllegalStateException("listResponse not mocked")
        }

        override suspend fun getTransaction(id: String): Response<SingleTransactionResponse> = throw UnsupportedOperationException()
    }

    private lateinit var dao: FakeTransactionDao
    private lateinit var apiService: FakeApiService
    private lateinit var repository: TransactionRepositoryImpl

    @Before
    fun setup() {
        dao = FakeTransactionDao()
        apiService = FakeApiService()
        repository = TransactionRepositoryImpl(apiService, dao)
    }

    @Test
    fun calculateSummary_aggregatesTotalsProperly() {
        val txList = listOf(
            TransactionEntity(
                id = "tx_1",
                provider = "MTN",
                amountMinorUnits = 5000L,
                currency = "GHS",
                flowType = "OUTBOUND",
                counterparty = "Kofi",
                referenceId = "ref_1",
                timestampMs = 1000L
            ),
            TransactionEntity(
                id = "tx_2",
                provider = "MTN",
                amountMinorUnits = 2500L,
                currency = "GHS",
                flowType = "OUTBOUND",
                counterparty = "ECG",
                referenceId = "ref_2",
                timestampMs = 2000L
            ),
            TransactionEntity(
                id = "tx_3",
                provider = "TELECEL",
                amountMinorUnits = 10000L,
                currency = "GHS",
                flowType = "INBOUND",
                counterparty = "Ama",
                referenceId = "ref_3",
                timestampMs = 3000L
            )
        )

        val summary = repository.calculateSummary(txList)

        assertEquals(7500L, summary.totalOutboundMinorUnits)
        assertEquals(10000L, summary.totalInboundMinorUnits)
        assertEquals(2500L, summary.netMinorUnits)
    }

    @Test
    fun refreshTransactions_storesRemotelyFetchedDataInDao() = runTest {
        val remoteDto = TransactionDto(
            id = "tx_remote_101",
            accountId = "acc_1",
            amountMinorUnits = 4500L,
            currency = "GHS",
            direction = "OUTBOUND",
            transactionType = "PAYMENT",
            counterparty = "Melcom Ghana",
            providerReference = "REF999888",
            status = "COMPLETED",
            userCategoryId = null,
            userNotes = "Groceries",
            eventTimestamp = "2026-10-13T10:00:00Z",
            createdAt = "2026-10-13T10:00:00Z"
        )

        apiService.listResponse = Response.success(
            ListTransactionsResponse(
                data = listOf(remoteDto),
                pagination = PaginationMetaDto(limit = 100, offset = 0, total = 1L)
            )
        )

        val result = repository.refreshTransactions()

        assertTrue(result.isSuccess)
        val entities = dao.getAllTransactions().first()
        assertEquals(1, entities.size)
        assertEquals("tx_remote_101", entities[0].id)
        assertEquals(4500L, entities[0].amountMinorUnits)
        assertEquals("Melcom Ghana", entities[0].counterparty)
    }
}
