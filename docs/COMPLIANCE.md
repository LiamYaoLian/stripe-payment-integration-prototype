# Compliance — PCI & Data Privacy

> Engineering notes for go-live planning. **Not legal advice** — have counsel review before production.

## PCI DSS scope

This prototype uses **Stripe Checkout** (hosted or embedded). Card data is collected and processed by Stripe; **this application never stores, processes, or transmits PAN/CVV**.

| Control | How this repo addresses it |
|---------|---------------------------|
| Card data | Stripe-hosted UI only; no Card Element / raw card fields |
| Stripe keys | `STRIPE_SECRET_KEY` server-side only; publishable key in frontend build |
| Webhooks | Signature verification on raw body (`STRIPE_WEBHOOK_SECRET`) |
| Logs | Structured logs; do not log secrets, tokens, or payment payloads |

**Typical merchant path:** eligible for **PCI SAQ A** (or SAQ A-EP if using embedded Checkout on your domain — confirm with Stripe + QSA). See [Stripe PCI guide](https://docs.stripe.com/security/guide).

## Data inventory

| Data | Stored where | Purpose | Retention (prototype default) |
|------|--------------|---------|-------------------------------|
| `customer_email` | `orders` | Checkout prefill, guest order history | Until you define a policy |
| Order line items | `order_items` | Receipt / support | Same as orders |
| `access_token_hash` | `orders` | Protect order reads | Same as orders |
| Stripe session / PI ids | `orders` | Webhook correlation, refunds | Same as orders |
| Webhook payloads | `webhook_events` | Idempotency / audit | Same as orders |
| Guest JWT | Client only | `GET /api/orders/mine` | 24h TTL |

**Not collected:** passwords, full payment methods, billing addresses (unless Stripe Checkout collects and you sync later).

## GDPR / privacy (baseline)

Before EU/UK traffic:

1. **Lawful basis** — document why you process email (contract / legitimate interest for order fulfilment).
2. **Privacy notice** — link from checkout; state Stripe as processor.
3. **Data subject requests** — define process to export/delete `orders` + `webhook_events` by email.
4. **Sub-processors** — Stripe DPA; hosting provider DPA.
5. **Breach playbooks** — who to notify; Stripe status page for payment incidents.

## Production checklist (compliance)

- [ ] Confirm SAQ type with Stripe integration model (hosted vs embedded)
- [ ] Privacy policy published; cookie notice if using analytics
- [ ] Data retention job (e.g. anonymize orders after N months)
- [ ] Restrict DB/backups access; TLS in transit (`sslmode=require` in prod)
- [ ] Annual access review for `STRIPE_SECRET_KEY`, `AUTH_JWT_SECRET`, DB credentials
