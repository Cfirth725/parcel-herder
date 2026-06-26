# 📦 Parcel Herder

A lightweight, concurrent logistics background daemon and dashboard built in Go. `parcel-herder` runs locally on a home lab node, scraping inbound shipping notifications via secure IMAP streams, managing carrier telemetry with an internal SQLite engine, and providing an elegant single-pane-of-glass terminal view for pending deliveries and smart locker pickup tokens.

## 🚀 Key Architectural Features
- **Multi-Tenant State Management:** Engineered with a data protection model allowing isolation across multiple discrete mailboxes without data cross-contamination.
- **Defensive Multi-Box Tracking:** Implements SQLite composite primary keys to safely track split-shipments sharing identical tracking metadata.
- **Locker Token Pool Optimization:** Minimizes visual friction by tracking multi-package deliveries via a single active master key row per account.
- **DevSecOps First Design:** Complete decoupling of configuration environments via `.env` architectures. No plain-text access tokens touch the database layer.

## 🛠️ Project File Layout
- `/cmd`: Application entry points and runtime initializers.
- `/internal`: Core internal system modules (Database, IMAP Parser, HTTP Engine).
- `/web`: Raw HTML templates and plain, lightweight CSS interfaces.

## 🗺️ Engineering Roadmap
### Phase 1: Local Foundation & Data Isolation 
- [ ] Establish secure `.env` runtime injection.
- [ ] Implement multi-tenant schema DDL via external `schema.sql`.
- [ ] Construct the dynamic `db.go` API layer supporting multi-user account allocation.

### Phase 2: Telemetry Ingestion Engine
- [ ] Build thread-safe `sync.WaitGroup` worker loops inside `internal/parser`.
- [ ] Establish regex boundary constraints to parse standard carriers (USPS, FedEx, UPS).
- [ ] Build strict Lookbehind Regex arrays to isolate anonymous 6-digit Amazon Hub Locker pickup tokens.

### Phase 3: Hardware Exception Catching
- [ ] Map driver-shortfall detection logic (detecting "Left at Locker" status flags lacking actual electronic Hub terminal verification codes).
- [ ] Build automated cron routines for database state-purging and archiving.

### Phase 4: Lightweight Delivery Interface
- [ ] Craft a high-contrast, vanilla CSS Grid dashboard layout with full semantic dark mode integration.
- [ ] Implement the single-stroke manual override "Picked Up!" action route.