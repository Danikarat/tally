# Tally — Automatic Expense Tracker for Ghana

[![Go Version](https://img.shields.io/badge/go-1.25%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Private-red.svg)](LICENSE)
[![GitHub Projects](https://img.shields.io/badge/project-Tally%20Kanban%20Board-success)](https://github.com/users/Danikarat/projects/2/views/1?system_template=kanban)

**Tally** is an Android-first personal expense-tracking application designed specifically for the Ghanaian financial ecosystem. The primary objective is to automatically capture and categorize financial transactions with minimal user input, primarily by parsing mobile money and banking transaction notifications (SMS, statements, and email).

---

## 📋 Project Management & Agile Board

All work on this codebase is organized into an Agile Kanban backlog tracked directly in GitHub Projects:

* **Kanban Board**: [https://github.com/users/Danikarat/projects/2/views/1](https://github.com/users/Danikarat/projects/2/views/1?system_template=kanban)
* **Project Owner**: `Danikarat`
* **Project Number**: `2`
* **Board Columns**: `Backlog` ➔ `Ready` ➔ `In progress` ➔ `In review` ➔ `Done`
* **Repository**: [`Danikarat/tally`](https://github.com/Danikarat/tally)

> 💡 **For AI Coding Agents & Developers**: Before picking up any task, read [`docs/AGENT_GUIDELINES.md`](docs/AGENT_GUIDELINES.md) for CLI commands, rules of engagement, and workflow steps.

---

## 🎯 Target Providers (Initial Scope)

1. **MTN MoMo** (Ghana)
2. **Telecel Cash** (formerly Vodafone Cash)
3. **AT Money** (formerly AirtelTigo Money)

*Future iterations will expand to Ghanaian retail banks, statement imports, and email integrations.*

---

## 🏗️ Architecture & Technology Stack

### Backend
* **Language & Runtime**: Go (modular monolith)
* **HTTP Routing**: [Chi v5](https://github.com/go-chi/chi) with structured `slog` logging and request tracking
* **Database**: PostgreSQL 16 with versioned migrations (`golang-migrate`)
* **Data Access**: [sqlc](https://sqlc.dev/) for type-safe query generation and `pgx/v5`
* **API Documentation**: OpenAPI / REST over HTTPS

### Android Client
* **Language & UI**: Kotlin 2.x, Jetpack Compose, Material 3
* **Local Persistence**: Room Database (offline-first candidate caching)
* **Background Sync**: WorkManager with exponential backoff & network constraints
* **Security**: Android Keystore (`EncryptedSharedPreferences`)
* **Ingestion Pipeline**: In-app SMS `BroadcastReceiver` adhering to Google Play financial SMS exceptions

---

## ⚖️ Core Invariants & Engineering Rules

Any contributor or AI agent modifying this codebase must adhere strictly to these rules:

1. **NEVER INVENT SMS FORMATS**:
   * Do not guess message syntax, sender headers, field order, or phrasing.
   * Parsers must be built strictly against real, consented, and redacted SMS fixtures in `testdata/fixtures/` and verified with table-driven Go tests.
2. **INTEGER MINOR UNITS FOR CURRENCY**:
   * Never use floating-point numbers (`float32`, `float64`) for money.
   * Amounts are represented in integer minor units (pesewas). E.g., `GH₵ 125.00` = `12500` minor units.
3. **UTC IN DATABASE, ACCRA IN UI**:
   * Store timestamps as UTC `timestamptz`.
   * Display dates and group monthly metrics using `Africa/Accra` timezone.
4. **THREE-TIER DEDUPLICATION**:
   * **Tier 1 (Source Event)**: SHA-256 payload hashing of incoming SMS/event.
   * **Tier 2 (Fingerprint)**: Hash of `(user_id, account_id, amount, direction, counterparty, time_window)`.
   * **Tier 3 (Reconciliation)**: Linking refunds, reversals, and withdrawal fee line items.
5. **TENANT ISOLATION & PRIVACY**:
   * Every database query for transactions must filter by authenticated `user_id`.
   * Never log raw SMS bodies, auth tokens, passwords, or phone numbers.
   * Comply with Ghana's Data Protection Act, 2012 (Act 843).

---

## 📁 Repository Layout

```text
tally/
├── cmd/
│   └── api/                  # Application entrypoint (main.go)
├── internal/
│   ├── accounts/             # Financial accounts & MoMo wallets
│   ├── analytics/            # Spending summaries & category aggregations
│   ├── auth/                 # Argon2id password hashing, JWTs, middleware
│   ├── budgets/              # Monthly budgets & threshold alerts
│   ├── categorization/       # Category taxonomy & user rules
│   ├── config/               # Environment & configuration loader
│   ├── db/                   # sqlc generated queries & db connection
│   ├── ingestion/            # POST /v1/transactions/ingest & payload deduplication
│   ├── integrations/         # External integrations
│   ├── jobs/                 # Asynchronous background tasks
│   ├── parsers/              # Provider parser registry & adapters
│   │   ├── mtn/              # MTN MoMo Ghana parser adapter
│   │   ├── telecel/          # Telecel Cash parser adapter
│   │   └── atmoney/          # AT Money parser adapter
│   ├── server/               # HTTP server, Chi routes, and middleware
│   ├── transactions/         # Normalized transaction queries & corrections
│   └── users/                # User profiles and GDPR/Act 843 data purge
├── db/
│   ├── migrations/           # Up/down SQL migrations
│   └── queries/              # sqlc SQL query definitions
├── docs/
│   └── AGENT_GUIDELINES.md   # Guidelines for AI agents and developers
├── resources/
│   └── Product.md            # Detailed product specification document
├── testdata/
│   └── fixtures/             # Real redacted SMS test fixtures
├── .env.example              # Documented environment template
├── docker-compose.yaml       # Local PostgreSQL 16 container definition
└── go.mod                    # Go module definition
```

---

## 🚀 Getting Started

### Prerequisites
* **Go**: 1.25+ installed (`go version`)
* **Docker & Docker Compose**: For running PostgreSQL locally
* **GitHub CLI (`gh`)**: For managing Project #2 issues and board items

### 1. Configure Environment
```bash
cp .env.example .env
```

### 2. Run the API Server
```bash
go run ./cmd/api
```
The server will boot on `http://localhost:8080`.

### 3. Verify Health Check
```bash
curl -i http://localhost:8080/healthz
```
Expected output:
```json
{"environment":"development","status":"ok"}
```

### 4. Run Unit Tests
```bash
go test -v -race ./...
```

---

## 🗺️ Implementation Milestones

* **Milestone 1: Foundation** (In Progress)
  * [x] **[#17](https://github.com/Danikarat/tally/issues/17)**: Initialize Go backend modular monolith skeleton, toolchain, and config
  * [ ] **[#18](https://github.com/Danikarat/tally/issues/18)**: Establish local development environment with Docker Compose for PostgreSQL
  * [ ] **[#22](https://github.com/Danikarat/tally/issues/22)**: Setup migration runner and create initial users and financial_accounts tables
  * [ ] **[#32](https://github.com/Danikarat/tally/issues/32)**: Initialize Android project with Kotlin, Jetpack Compose, Material 3
* **Milestone 2: First Transaction Ingestion**
  * Source-event and transaction schemas, ingestion endpoint, idempotency, real MTN MoMo SMS sample collection ([#28](https://github.com/Danikarat/tally/issues/28)), and fixture-driven parser.
* **Milestone 3: Android Integration**
  * Room database persistence, WorkManager synchronization, SMS receiver, and end-to-end integration test.
* **Milestone 4: Financial Correctness**
  * Deterministic classification, transfers vs fees, refunds, reversals, multi-tier deduplication.
* **Milestone 5: Provider Expansion**
  * Telecel Cash and AT Money integrations backed by verified fixtures.
* **Milestone 6: Product Completion**
  * Category budgets, analytics dashboard, security audit, Ghana Act 843 compliance, Google Play release.
