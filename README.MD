# 📦 Parcel Herder

A lightweight, concurrent logistics background daemon and dashboard built in Go. `parcel-herder` runs locally on a home lab node, scraping inbound shipping notifications via secure IMAP streams, managing carrier telemetry with an internal SQLite engine, and providing an elegant single-pane-of-glass dashboard view for pending deliveries and smart locker pickup tokens.

## ✨ Key Architectural Features
- **Multi-Tenant State Management:** Engineered with a data protection model allowing isolation across multiple discrete mailboxes without data cross-contamination.
- **Cryptographic Blind Indexing:** Enforces a DevSecOps-first approach by storing SHA-256 hashes of incoming emails, keeping raw personal identities completely off the disk layout.
- **Fail-Fast Configuration Infrastructure:** Enforces strict environment boundaries at initialization, crashing defensively with clear logs if `PORT` bindings or security credentials are missing.
- **Conditional Upsert Ingestion:** Distinguishes between standard tracking barcodes (deduplicated automatically on conflict) and platform-encrypted shipments (appended sequentially as actionable alerts).
- **Locker Token Consolidation & Atomic Cascades:** Pins a unified master locker key console at the top of the viewport. Deactivating the master code triggers an atomic database transaction that automatically archive-sweeps all unverified locker shortfall packages simultaneously.
- **Smart Link Resolver:** Computes dynamic, native deep-links client-side for **USPS, FedEx, UPS, DHL, and OSM Worldwide**, completely leapfrogging commercial email marketing redirect servers to preserve privacy.
- **Urgency-Driven UI Filtering:** Leverages an internal SQLite temporal state evaluation machine to automatically highlight and pin items marked `ARRIVING TODAY` or flag stale rows for manual review.
- **Lightweight High-Contrast Presentation:** Rejects bloated front-end frameworks in favor of semantic HTML5, native CSS variables, and a distraction-free dark mode theme optimized for local terminal dashboards.
- **Structured DevOps Telemetry:** Built with consistent, scannable log tokens (`[INIT]`, `[SECURE]`, `[SYNC]`, `[SERVER]`, `[OK]`, `[ERROR]`, `[SHUTDOWN]`) for clean, production-grade terminal visibility.
- **Graceful Shutdown Fallbacks:** Intercepts system interruption signals (`Ctrl+C` / `SIGTERM`) via native channels, executing an explicit database `wal_checkpoint(TRUNCATE)` to cleanly flush memory-mapped journals back to disk before connection severance.
- **Asynchronous Data Lifecycles:** Runs dedicated background workers using native channel tickers to manage internal disk optimization and state auditing without blocking main server processing loops.

## 🛠️ Project File Layout
```text
├── cmd/
│   └── main.go                 # App entry point, multi-tenant sync loop, background workers, & server init
├── internal/
│   ├── db/
│   │   ├── db.go               # Thread-safe SQLite atomic transaction routines targets
│   │   └── schema.sql          # Normalized multi-table schemas (accounts, packages, locker_status)
│   ├── parser/
│   │   └── parser.go           # Compile-once regex logistics telemetry engine & closed-system bypass filters
│   ├── scraper/
│   │   └── scraper.go          # Secure TLS IMAP client & message extraction loop
│   └── server/
│       └── server.go           # HTTP routing engine, action handlers, background tickers, & data auditing
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
- [x] Extend regex engine footprints to support international DHL Express and unified OSM Worldwide tracking.
- [x] Build strict pattern arrays to isolate anonymous 6-digit Amazon Hub Locker pickup tokens.
- [x] Construct a secure TLS IMAP connection layer utilizing `go-imap/v2` streaming mechanics.
- [x] Connect the scraper's parsing matches directly to database write-routines (`InsertOrUpdatePackage`).
- [x] Establish isolated native test coverage using built-in `go test` mechanics.

### Phase 3: Lightweight Delivery Interface & Routing
- [x] Craft a high-contrast, vanilla CSS Grid dashboard layout with full semantic dark mode integration.
- [x] Refactor database querying into distinct parallel tracks to isolate global locker state metrics from individual item collections.
- [x] Implement explicit port checking and fail-fast environment guardrails on application boot.
- [x] Integrate a functional manual management sidebar module for off-stream package registry overrides.

### Phase 4: State Machine Refinement & Human-in-the-Loop Controls
- [x] Implement an explicit state-toggle mechanism (HTTP POST route) to manually clear or deactivate active locker codes.
- [x] Engineer an atomic SQL transaction block to cascade-archive unverified building locker packages when the master PIN is cleared.
- [x] Build multi-option client-side sync controls directly onto package cards (`✓ Arrived`, `📍 Locker`, and `+1 Day Delay`).
- [x] Implement a conditional carrier-conversion toggle shortcut to instantly upgrade shipping partner handoffs over to native USPS tracking.
- [x] Integrate defensive edge-case interceptors for encrypted, closed-ecosystem notifications (**Etsy**, **Shop App**, and **Amazon Logistics**).

### Phase 5: Automation Loops & Performance Tuning
- [x] Map same-day driver-shortfall detection logic (`CheckDriverShortfallsLocal` to audit missing locker codes against active trajectories).
- [x] Build automated background cron loops for a 12-hour database vacuuming and archiving purge cycle.
- [x] Implement a signal-trapping graceful shutdown sequence to ensure clean disk states on close.
- [ ] Upgrade the polling engine to use IMAP IDLE connection sockets for near-zero-latency ingestion.