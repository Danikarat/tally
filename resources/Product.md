# Project Context: Automatic Expense Tracker for Ghana

## 1. Project Overview

I am building an Android-first personal expense-tracking application using **Go and PostgreSQL** for the backend.

The primary goal is to automatically track users' expenses over time with minimal manual input. Many financial transactions generate SMS or email notifications, including mobile money transfers, bank transactions, withdrawals, payments, and transaction fees. The application should use these notifications as a source of transaction data.

The initial target market is Ghana.

### Initial mobile money providers

1. MTN MoMo
2. Telecel Cash
3. AT Money (formerly AirtelTigo Money)

The application may support local banks and email-based transaction notifications in later iterations.

## 2. Product Priorities

The most important product priority is **automatic transaction tracking with minimal user intervention**.

The application should:

- Detect supported transaction notifications.
- Extract transaction details automatically.
- Normalize transactions into a common data model.
- Categorize transactions automatically where possible.
- Avoid duplicate transactions.
- Distinguish expenses from income, transfers, refunds, withdrawals, fees, and failed transactions.
- Work offline and synchronize data when connectivity returns.
- Allow users to correct transaction details and categories.
- Provide spending summaries, category breakdowns, budgets, and historical trends.

Accuracy is more important than pretending to support every transaction format.

## 3. Proposed Technology Stack

### Android application

- Kotlin
- Jetpack Compose
- Room for local persistence
- WorkManager for deferred synchronization
- Android Keystore-backed storage for sensitive credentials
- Android SMS integration, subject to the applicable Android and Google Play permission requirements

The Android client should support local persistence and synchronization retries.

### Backend

- Go
- Chi or Gin for HTTP routing
- pgx for PostgreSQL connectivity
- sqlc for type-safe SQL access
- OpenAPI for API documentation

Start with a modular monolith. Do not introduce microservices unless a demonstrated need justifies them.

### Database

- PostgreSQL
- Versioned database migrations
- Constraints and indexes
- Idempotent transaction ingestion

## 4. SMS Collection Strategy

SMS collection is the immediate engineering priority.

A phone number alone does not grant access to its SMS inbox. The application must operate on a device or use another explicitly authorized source.

The intended flow is:

1. The user installs the Android app.
2. The app explains its SMS collection functionality and obtains applicable permissions.
3. An Android component receives supported incoming transaction messages.
4. A provider-specific parser identifies the message format.
5. The parser extracts structured transaction information locally where practical.
6. The app stores a transaction candidate in Room.
7. A background synchronization worker uploads the candidate to the Go backend.
8. The backend validates and deduplicates the candidate.
9. PostgreSQL stores the normalized transaction.
10. The dashboard updates after successful synchronization.

### Android permission and distribution constraints

`READ_SMS` and `RECEIVE_SMS` are restricted permissions under Google Play policy. SMS-based money management may qualify for an exception, but approval must not be assumed.

Before depending on these permissions for public distribution, verify the current Google Play policy, eligibility requirements, permission declarations, disclosures, and review process.

An Android `NotificationListenerService` may be considered as a supplementary source, but it is not equivalent to SMS inbox access and cannot reliably recover historical messages.

SMS Retriever is not a general solution for collecting bank or mobile money messages.

If SMS permission approval is unavailable, retain alternative ingestion methods such as statement imports, authorized email integrations, and other permitted data sources.

## 5. Providers and Parser Architecture

The first three providers are:

- MTN MoMo
- Telecel Cash
- AT Money

Each provider should have its own parser adapter implementing a shared interface.

Illustrative Go interface:

```go
type TransactionParser interface {
    Provider() string
    CanParse(sender, body string) bool
    Parse(sender, body string) (TransactionCandidate, error)
}
```

The exact implementation can be refined during development.

The parser registry should select the appropriate adapter based on reliable sender identification and message-format evidence.

### Critical requirement: Do not invent SMS formats

We have not yet collected actual transaction SMS examples from the three providers.

Do not assume that message wording, sender IDs, field names, transaction references, or number formats are known.

The next immediate task is to obtain real, consented examples from each provider, with personal information redacted. Preserve the original wording, punctuation, field order, and number formatting.

Build table-driven Go tests from these fixtures before claiming that any provider is supported.

### Transaction message types to cover

For each provider, test:

- Money sent
- Money received
- Merchant or bill payments
- Cash withdrawals
- Transaction fees
- Reversals and refunds
- Failed transactions
- Pending transactions
- Balance notifications without a transaction
- Duplicate messages
- Unrecognized message formats

Parsers should return an explicit error, unknown state, or review-required result when information is insufficient. Never invent missing values.

## 6. Normalized Transaction Model

A transaction candidate should support fields such as:

- Provider
- Source type
- Sender identifier
- Currency
- Amount
- Direction
- Transaction type
- Counterparty or merchant
- Provider transaction reference
- Transaction status
- Event timestamp
- Parser version
- Parsing status
- Source-event identifier

Represent monetary values as integer minor units or another explicitly precise monetary representation. Do not use floating-point values for financial calculations.

For Ghanaian cedi transactions, represent GH₵125.00 as 12500 minor units, with the currency exponent defined consistently.

Store event timestamps using PostgreSQL `timestamptz`, and display dates using the `Africa/Accra` timezone.

Distinguish these transaction types:

- Expense
- Income
- Transfer
- Refund
- Fee
- Cash withdrawal
- Unknown or unclassified

An incoming transfer must not automatically be classified as income. A transfer between accounts owned by the same user must not inflate spending. Refunds and reversals must be linked to the relevant transactions where possible.

## 7. Proposed Backend Structure

```text
expense-tracker/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── auth/
│   ├── users/
│   ├── accounts/
│   ├── ingestion/
│   ├── parsers/
│   │   ├── registry.go
│   │   └── providers/
│   ├── transactions/
│   ├── categorization/
│   ├── budgets/
│   ├── analytics/
│   ├── integrations/
│   └── jobs/
├── db/
│   ├── migrations/
│   └── queries/
├── api/
│   └── openapi.yaml
└── go.mod
```

This is a proposed structure, not a claim that these files or modules already exist.

Inspect the existing repository before creating files. Adapt the implementation to the current project structure if development has already started.

## 8. Proposed PostgreSQL Tables

### `users`

Stores user accounts and authentication metadata.

### `financial_accounts`

Represents mobile money wallets, bank accounts, cash, and cards. Avoid requiring full account numbers unless absolutely necessary.

### `transactions`

Stores normalized transaction records, including:

- User and account IDs
- Amount in minor units
- Currency
- Direction and transaction type
- Merchant or counterparty
- Category
- Event timestamp
- Transaction status
- Transfer grouping or linkage

### `source_events`

Tracks incoming SMS, email, and imported transaction candidates, including:

- User ID
- Source type
- Provider
- External reference where available
- Payload hash
- Received timestamp
- Associated transaction ID
- Parse status

### `categories` and `budgets`

Store user-defined categories, classification rules, budgets, and spending limits.

### `integrations` and `sync_cursors`

Support future email connections, synchronization checkpoints, and encrypted integration tokens.

Use foreign keys, indexes, and uniqueness constraints where appropriate.

## 9. Deduplication and Reliability

Deduplication is essential.

Use three levels:

1. Source-event deduplication: prevent the same SMS or email from being processed repeatedly.
2. Transaction deduplication: recognize the same underlying payment received from multiple sources.
3. Business-level reconciliation: link refunds, reversals, and paired transfers to existing transactions.

Prefer stable provider transaction references when available.

When references are unavailable, use a carefully designed fingerprint based on user, account, amount, currency, direction, event time, and counterparty. Treat ambiguous matches as potential duplicates requiring additional checks rather than automatically merging all identical purchases.

Enforce idempotency at the database level where reliable keys exist.

The Android app should retain unsynchronized candidates locally and retry uploads safely.

## 10. Proposed API Endpoints

- `POST /v1/auth/register`
- `POST /v1/auth/login`
- `POST /v1/transactions/ingest`
- `GET /v1/transactions`
- `PATCH /v1/transactions/{id}`
- `GET /v1/analytics/summary`
- `GET /v1/analytics/categories`
- `GET /v1/budgets`
- `POST /v1/integrations/email/connect`

The ingestion endpoint should support batched candidates, authentication, validation, idempotency, and useful error responses.

These endpoints are proposed and may be changed to fit the actual repository.

## 11. Security and Privacy

Financial SMS messages may contain sensitive personal and financial information.

Requirements:

- Explicit user consent and clear disclosures.
- Least-privilege access.
- Local parsing where practical.
- Upload structured transaction fields rather than full SMS bodies by default.
- HTTPS for all network communication.
- Secure authentication and token revocation.
- Android Keystore-backed credential storage.
- Encryption of sensitive integration tokens at rest.
- Strict per-user authorization on every transaction operation.
- No secrets, tokens, or full SMS bodies in application logs.
- User-controlled integration disconnection and data deletion.
- Minimal retention of raw source data.
- Tests proving that users cannot access each other's transactions.

Investigate applicable Ghanaian data-protection requirements, including the Data Protection Act, 2012 (Act 843), Data Protection Commission registration obligations, and any requirements applicable to the product's processing and hosting arrangements.

## 12. Initial Implementation Milestones

### Milestone 1: Backend and database

Create the Go API, PostgreSQL schema, migrations, authentication, financial accounts, and basic transaction CRUD.

### Milestone 2: Ingestion and parsers

Build normalized transaction candidates, provider parser interfaces, deduplication, and fixture-based tests.

### Milestone 3: Android application

Implement onboarding, local persistence, transaction list, dashboard, and background synchronization.

### Milestone 4: SMS integration

Validate actual transaction messages, implement the first supported provider adapter, and address Android and Google Play permission requirements.

### Milestone 5: Categorization and analytics

Implement category rules, user corrections, refunds, transfers, spending summaries, and budgets.

### Milestone 6: Pilot and release readiness

Test on real Android devices, verify parser accuracy, audit security, and prepare distribution and privacy documentation.

The proposed schedule is approximately four weeks for a focused initial implementation, but external approval and access to representative provider messages may take longer.

## 13. Immediate Next Task

**Focus exclusively on making SMS collection and parsing reliable before expanding the product.**

Proceed in this order:

1. Inspect the existing project repository and identify what has already been implemented.
2. Confirm the Android app language, framework, build configuration, and current permission setup.
3. Obtain one real, consented MTN MoMo transaction SMS with personal information redacted.
4. Create a Go test fixture that asserts the extracted fields and transaction classification.
5. Implement the first parser against that fixture.
6. Add tests for duplicates, failed payments, refunds, transfers, fees, and malformed messages.
7. Repeat for Telecel Cash.
8. Repeat for AT Money.
9. Connect the Android collector to the Go ingestion API.
10. Test offline synchronization, repeated uploads, and permission denial.

Do not claim a provider parser is complete until it passes tests based on real message formats and relevant edge cases.

## 14. Working Instructions for the Coding Agent

- Work on one clearly defined task at a time.
- Inspect the repository before changing code.
- Do not assume that proposed files, endpoints, schemas, or modules already exist.
- Do not invent provider-specific SMS patterns.
- Prefer deterministic parsing and table-driven tests over LLM-based extraction.
- Return explicit errors for unsupported or ambiguous messages.
- Use precise monetary arithmetic.
- Keep the implementation simple and maintainable.
- Run relevant tests after every meaningful change.
- Report files changed, commands executed, test results, and unresolved issues.
- Do not expand into email integrations, advanced AI categorization, or microservices while the SMS foundation is still incomplete.

**First deliverable:** a tested Go parser for one real MTN MoMo SMS format, followed by a repeatable process for adding Telecel Cash and AT Money.
