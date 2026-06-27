# 📦 Parcel Herder

A lightweight, concurrent logistics background daemon and dashboard built in Go. `parcel-herder` runs locally on a home lab node, scraping inbound shipping notifications via secure IMAP streams, managing carrier telemetry with an internal SQLite engine, and providing an elegant single-pane-of-glass terminal view for pending deliveries and smart locker pickup tokens.

## ✨ Key Architectural Features
- **Multi-Tenant State Management:** Engineered with a data protection model allowing isolation across multiple discrete mailboxes without data cross-contamination.
- **Cryptographic Blind Indexing:** Enforces a DevSecOps-first approach by storing SHA-256 hashes of incoming emails, keeping raw personal identities completely off the disk layout.
- **Fail-Fast Configuration Infrastructure:** Enforces strict environment boundaries at initialization, crashing defensively with clear logs if `PORT` bindings or security credentials are missing.
- **Conditional Upsert Ingestion:** Distinguishes between standard tracking barcodes (deduplicated automatically on conflict) and platform-encrypted shipments (appended sequentially as actionable alerts).
- **Locker Token Consolidation:** Pins a unified master locker key console at the top of the viewport, aligning with physical smart hubs that aggregate multiple incoming shipments behind a single active PIN.
- **Lightweight High-Contrast Presentation:** Rejects bloated front-end frameworks in favor of semantic HTML5, native CSS variables, and a distraction-free dark mode theme optimized for local terminal dashboards.
- **Structured DevOps Telemetry:** Built with consistent, scannable log tokens (`[INIT]`, `[SECURE]`, `[SYNC]`, `[SERVER]`, `[OK]`, `[ERROR]`) for clean, production-grade terminal visibility.

## 🛠️ Project File Layout
```text
├── cmd/
│   └── main.go                 # App entry point, multi-tenant sync loop, & web server init
├── internal/
│   ├── db/
│   │   ├── db.go               # Thread-safe SQLite atomic transaction routines
│   │   ├── db_test.go          # Native unit tests using fast, isolated :memory: targets
│   │   └── schema.sql          # Normalized multi-table schemas (accounts, packages, locker_status)
│   ├── parser/
│   │   └── parser.go           # Compile-once regex logistics telemetry engine
│   ├── scraper/
│   │   └── scraper.go          # Secure TLS IMAP client & message extraction loop
│   └── server/
│       └── server.go           # HTTP routing engine, asset mounting, & separate stream queries
├── web/
│   ├── static/
│   │   └── style.css           # High-contrast palette variables & grid layout rules
│   └── templates/
│       └── dashboard.html      # Responsive template with master locker console & package streams
├── .env                        # Local environment credentials (gitignored)
└── go.mod
```

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
- [x] Connect the scraper's parsing matches directly to database write-routines (`InsertOrUpdatePackage`).
- [x] Establish isolated native test coverage using built-in `go test` mechanics.

### Phase 3: Lightweight Delivery Interface & Routing
- [x] Craft a high-contrast, vanilla CSS Grid dashboard layout with full semantic dark mode integration.
- [x] Refactor database querying into distinct parallel tracks to isolate global locker state metrics from individual item collections.
- [x] Implement explicit port checking and fail-fast environment guardrails on application boot.
- [x] Integrate a functional manual management sidebar module for off-stream package registry overrides.

### Phase 4: State Machine Refinement & Hardware Verification
- [ ] Implement an explicit state-toggle mechanism (HTTP POST route) to manually clear or mark the `ACTIVE AMAZON LOCKER MASTER HUB PIN` as inactive once lockers are empty.
- [ ] Map driver-shortfall detection logic (`CheckDriverShortfalls` to identify "Left at Hub/Locker" string states that slipped through without an accompanying electronic code entry).
- [ ] Build automated cron routines for database state-purging and archiving.