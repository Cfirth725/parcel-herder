# 📦 Parcel Herder

A lightweight, concurrent logistics background daemon and dashboard built in Go. `parcel-herder` runs locally on a home lab node, scraping inbound shipping notifications via secure IMAP streams, managing carrier telemetry with an internal SQLite engine, and providing an elegant single-pane-of-glass terminal view for pending deliveries and smart locker pickup tokens.

## Key Architectural Features
- **Multi-Tenant State Management:** Engineered with a data protection model allowing isolation across multiple discrete mailboxes without data cross-contamination.
- **Cryptographic Blind Indexing:** Enforces a DevSecOps-first approach by storing SHA-256 hashes of incoming emails, keeping raw personal identities completely off the disk layout.
- **Defensive Multi-Box Tracking:** Implements SQLite composite primary keys to safely track split-shipments sharing identical tracking metadata.
- **Locker Token Pool Optimization:** Minimizes visual friction by tracking multi-package deliveries via a single active master key row per account.
- **DevSecOps First Design:** Complete decoupling of configuration environments via `.env` architectures. No plain-text access tokens touch the database layer.

## 🛠️ Project File Layout
```text
├── cmd/
│   └── main.go                 # App entry point & runtime initialization loop
├── internal/
│   ├── db/
│   │   ├── db.go               # Thread-safe SQLite connection & blind index routines
│   │   └── schema.sql          # Multi-tenant data plane schema definitions
│   ├── parser/
│   │   └── parser.go           # Compile-once regex logistics telemetry engine
│   └── scraper/
│       └── scraper.go          # Secure TLS IMAP client & message extraction loop
├── web/
│   └── (HTML & plain CSS interfaces)
├── .env                        # Local environment credentials (gitignored)
└── go.mod
````

## 🗺️ Engineering Roadmap
### Phase 1: Local Foundation & Data Isolation
- [x] Establish secure `.env` runtime injection.
- [x] Implement multi-tenant schema DDL via external `schema.sql`.
- [x] Construct the dynamic `db.go` API layer supporting multi-user account allocation.
- [x] Secure the database layer using SHA-256 blind indices to anonymize target user mailboxes.

### Phase 2: Telemetry Ingestion Engine
- [x] Establish regex boundary constraints to parse standard carriers (USPS, FedEx, UPS).
- [x] Extend regex engine footprints to support international DHL Express and OSM Worldwide tracking.
- [x] Build strict pattern arrays to isolate anonymous 6-digit Amazon Hub Locker pickup tokens.
- [x] Construct a secure TLS IMAP connection layer utilizing `go-imap/v2` streaming mechanics.
- [ ] Connect the scraper's parsing matches directly to database write-routines (`CreatePackage`).

### Phase 3: Hardware Exception Catching
- [ ] Map driver-shortfall detection logic (detecting "Left at Locker" status flags lacking actual electronic Hub terminal verification codes).
- [ ] Build automated cron routines for database state-purging and archiving.

### Phase 4: Lightweight Delivery Interface
- [ ] Craft a high-contrast, vanilla CSS Grid dashboard layout with full semantic dark mode integration.
- [ ] Implement the single-stroke manual override "Picked Up!" action route.