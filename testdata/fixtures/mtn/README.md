# MTN MoMo Ghana SMS Fixtures

This directory contains real, consented, and sanitized MTN MoMo Ghana SMS message fixtures used for golden-file parser tests.

## Structure

Each fixture is a JSON file with the following format:

```json
{
  "id": "mtn_001_transfer_sent",
  "sender": "MobileMoney",
  "raw_body": "Payment of GHS 50.00 made to John Doe 0241234567. Current Balance: GHS 150.00. Reference: Groceries. Transaction ID: 12345678901. Fee was GHS 0.50.",
  "expected": {
    "provider": "MTN",
    "amount_minor_units": 5000,
    "currency": "GHS",
    "direction": "OUTBOUND",
    "transaction_type": "TRANSFER",
    "counterparty": "John Doe",
    "counterparty_identifier": "0241234567",
    "provider_reference": "12345678901",
    "balance_minor_units": 15000,
    "fee_minor_units": 50
  }
}
```

## Required Core Transaction Types

1. **Money Sent (P2P Transfer)**
2. **Money Received**
3. **Cash Out / Withdrawal at Agent**
4. **Bill / Pay Merchant Payment**
5. **Fee / E-Levy Notification**
