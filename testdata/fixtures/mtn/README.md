# MTN MoMo Ghana SMS Fixtures

This directory contains real, consented, and sanitized MTN MoMo Ghana SMS message fixtures used for golden-file parser tests.

## Structure

Each fixture is a JSON file with the following format:

```json
{
  "id": "mtn_001_transfer_sent",
  "sender": "MobileMoney",
  "raw_body": "Payment made for GHS 35.00 to KOFI MENSAH Current Balance: GHS 22.91 . Available Balance: GHS 22.91. Reference: Lunch. Transaction ID: 91383000001. Fee charged: GHS0.50 Tax charged: 0. Download the MoMo App for a Faster & Easier Experience. Click here: https://mtnmymomo.onelink.me/XJOt/MoMo",
  "expected": {
    "provider": "MTN",
    "amount_minor_units": 3500,
    "currency": "GHS",
    "direction": "OUTBOUND",
    "transaction_type": "TRANSFER",
    "counterparty": "KOFI MENSAH",
    "counterparty_identifier": null,
    "provider_reference": "91383000001",
    "balance_minor_units": 2291,
    "fee_minor_units": 50
  }
}
```

## Inventory of Verified Real Fixtures

| Fixture File | Type | Direction | Provider Ref | Real Scenario |
| :--- | :--- | :--- | :--- | :--- |
| `mtn_001_transfer_sent.json` | `TRANSFER` | `OUTBOUND` | `91383000001` | P2P / Merchant Payment Made |
| `mtn_002_payment_received.json` | `TRANSFER` | `INBOUND` | `91377000002` | Inter-network Transfer Received (Vodafone -> MTN) |
| `mtn_003_cash_in_received.json` | `TRANSFER` | `INBOUND` | `91240000003` | Cash In / Deposit Received at Agent |
| `mtn_004_bill_payment.json` | `EXPENSE` | `OUTBOUND` | `91333000004` | Utility / Invoice Bill Payment (`Bills.INV`) |
| `mtn_005_bundle_payment.json` | `EXPENSE` | `OUTBOUND` | `91338000005` | Data / Airtime Bundle Purchase (`MTN BUNDLE`) |
| `mtn_006_cash_out.json` | `CASH_WITHDRAWAL` | `OUTBOUND` | `91352000006` | Cash Out Withdrawal at Agent |
