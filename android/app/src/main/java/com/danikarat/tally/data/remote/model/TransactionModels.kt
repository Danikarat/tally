package com.danikarat.tally.data.remote.model

import com.google.gson.annotations.SerializedName

/**
 * TransactionDto represents the transaction payload from the Go backend.
 */
data class TransactionDto(
    @SerializedName("id") val id: String,
    @SerializedName("account_id") val accountId: String,
    @SerializedName("amount_minor_units") val amountMinorUnits: Long,
    @SerializedName("currency") val currency: String,
    @SerializedName("direction") val direction: String,
    @SerializedName("transaction_type") val transactionType: String,
    @SerializedName("counterparty") val counterparty: String? = null,
    @SerializedName("counterparty_identifier") val counterpartyIdentifier: String? = null,
    @SerializedName("provider_reference") val providerReference: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("user_category_id") val userCategoryId: String? = null,
    @SerializedName("user_notes") val userNotes: String? = null,
    @SerializedName("event_timestamp") val eventTimestamp: String,
    @SerializedName("created_at") val createdAt: String
)

/**
 * PaginationMetaDto contains pagination metadata.
 */
data class PaginationMetaDto(
    @SerializedName("limit") val limit: Int,
    @SerializedName("offset") val offset: Int,
    @SerializedName("total") val total: Long
)

/**
 * ListTransactionsResponse represents response from GET /v1/transactions.
 */
data class ListTransactionsResponse(
    @SerializedName("data") val data: List<TransactionDto>,
    @SerializedName("pagination") val pagination: PaginationMetaDto
)

/**
 * SingleTransactionResponse represents response from GET /v1/transactions/{id}.
 */
data class SingleTransactionResponse(
    @SerializedName("transaction") val transaction: TransactionDto
)
