# Instructions & Guidelines for AI Coding Agents

Welcome to **Tally** (`Danikarat/tally`). This document provides the operational guidelines, project board configuration, and engineering rules that every AI coding assistant (or new developer) must follow when working in this repository.

---

## 1. Quick Project Context

* **Product**: Automated, privacy-first personal expense-tracking application for Ghana.
* **Target Providers**: MTN MoMo, Telecel Cash, AT Money.
* **Backend**: Go (modular monolith), Chi v5, PostgreSQL 16, `pgx/v5` + `sqlc`.
* **Android**: Kotlin, Jetpack Compose, Material 3, Room, WorkManager, Android Keystore.
* **Specification Document**: Full product requirements are detailed in [`resources/Product.md`](../resources/Product.md).

---

## 2. GitHub Project & Kanban Board

All features, tasks, and bugs are tracked in GitHub Projects:

* **Board URL**: [https://github.com/users/Danikarat/projects/2/views/1](https://github.com/users/Danikarat/projects/2/views/1?system_template=kanban)
* **Project Owner**: `Danikarat`
* **Project Number**: `2`
* **Project ID**: `PVT_kwHOB42iUc4Bmg7i`
* **Repository**: [`Danikarat/tally`](https://github.com/Danikarat/tally)

### Board Columns & Status Field Options
* **`Backlog`**: Upcoming issues, blocked by dependencies or earlier milestones.
* **`Ready`**: Fully unblocked tickets ready for immediate execution.
* **`In progress`**: Currently active ticket (work on **one ticket at a time**).
* **`In review`**: Work completed and waiting for human review.
* **`Done`**: Accepted, tested, merged, and closed.

---

## 3. GitHub CLI (`gh`) Commands Cheatsheet

Always use these commands to manage the board directly from the shell:

### Check Project Items
```bash
gh project item-list 2 --owner Danikarat --limit 100
```

### View Issue Details
```bash
gh issue view <ISSUE_NUMBER> --repo Danikarat/tally
```

### Move Ticket to `In progress`
```bash
gh project item-edit 2 --owner Danikarat --url https://github.com/Danikarat/tally/issues/<NUM> --field "Status" --value "In progress"
```

### Move Ticket to `Done`
```bash
gh project item-edit 2 --owner Danikarat --url https://github.com/Danikarat/tally/issues/<NUM> --field "Status" --value "Done"
```

### Close Issue with Completion Summary
*(Always use single quotes around multiline comments to prevent bash backtick interpolation)*
```bash
gh issue close <NUM> --repo Danikarat/tally --comment 'Completed in commit <HASH>.
### Summary:
- Implemented ...
- Tests pass: go test -race ./...
'
```

---

## 4. Standard Procedure for Executing a Ticket

When the user asks to work on a ticket or start the next task, follow this exact sequence:

1. **Inspect Repo & Git Status**:
   ```bash
   git status
   git log -n 3
   ```
2. **Review the Ticket**:
   Read the issue description, acceptance criteria, definition of done, and dependencies (`gh issue view <NUM> --repo Danikarat/tally`).
3. **Move Ticket to `In progress`**:
   Update status in Project #2.
4. **Implement Cleanly**:
   * Inspect existing code before writing new code.
   * Preserve documentation and comments.
   * Write unit tests alongside implementation.
5. **Verify & Test**:
   ```bash
   gofmt -s -w .
   go test -v -race ./...
   ```
6. **Commit & Push**:
   ```bash
   git add <FILES>
   git commit -m "feat(<scope>): description (#<NUM>)"
   git push origin main
   ```
7. **Complete Ticket**:
   * Move ticket to `Done` in Project #2.
   * Close the issue on GitHub.
   * Report changed files, test results, and next recommended ticket to the user.

---

## 5. Non-Negotiable Engineering Invariants

### ❌ Never Invent SMS Formats
We do not guess SMS text, headers, field order, or punctuation.
* Parsers must be tested exclusively against real, consented, and redacted SMS messages stored in `testdata/fixtures/<provider>/`.
* If a message cannot be parsed with certainty, return an explicit error (e.g. `ErrUnrecognizedFormat`), never invent missing fields.

### 💰 Minor Units for Money (Integer Pesewas)
* **Never use floating point numbers (`float32`, `float64`) for financial arithmetic.**
* Store all amounts as `int64` minor units.
* Ghanaian Cedis have 2 decimal places: `GH₵ 125.00` = `12500` minor units.

### 🕒 Timezones
* Store timestamps in database as UTC `TIMESTAMPTZ`.
* Display dates and group monthly spending using the `Africa/Accra` timezone.

### 🔒 Tenant Isolation & Privacy
* Every query accessing accounts or transactions must enforce `WHERE user_id = $1`.
* **Zero Logging of Sensitive Data**: Never log passwords, auth tokens, full SMS bodies, or phone numbers in application logs.
* Comply with Ghana's Data Protection Act, 2012 (Act 843).

---

## 6. Current Implementation Milestones

* **Milestone 1: Foundation**
  * [#17](https://github.com/Danikarat/tally/issues/17): Initialize Go modular monolith skeleton, toolchain, config (**DONE**)
  * [#18](https://github.com/Danikarat/tally/issues/18): Docker Compose for PostgreSQL 16 (**READY**)
  * [#22](https://github.com/Danikarat/tally/issues/22): Migration runner & initial users / financial_accounts tables (**READY**)
  * [#32](https://github.com/Danikarat/tally/issues/32): Android project foundation with Jetpack Compose (**READY**)
* **Milestone 2: First Transaction Ingestion**
  * [#28](https://github.com/Danikarat/tally/issues/28): Collect real redacted MTN MoMo SMS fixtures (**READY - Data Prerequisite**)
  * [#29](https://github.com/Danikarat/tally/issues/29): Define `TransactionParser` adapter interface
  * [#30](https://github.com/Danikarat/tally/issues/30): MTN MoMo parser implementation & table-driven tests
  * [#25](https://github.com/Danikarat/tally/issues/25): Batched transaction ingestion API (`POST /v1/transactions/ingest`)
* **Milestone 3: Android Integration**
* **Milestone 4: Financial Correctness**
* **Milestone 5: Provider Expansion (Telecel Cash & AT Money)**
* **Milestone 6: Product Completion & Release**
