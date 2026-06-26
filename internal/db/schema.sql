-- 1. Scalable Multi-Mailbox Accounts Tracker
CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL
);

-- 2. Multi-Box Split Shipment Package Table with Composite Unique Key
CREATE TABLE IF NOT EXISTS packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    tracking_number TEXT NOT NULL,
    box_sequence INTEGER DEFAULT 1,
    carrier TEXT NOT NULL,
    last_status TEXT NOT NULL,
    location_state TEXT NOT NULL,
    is_active INTEGER DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(account_id) REFERENCES accounts(id),
    UNIQUE(tracking_number, box_sequence)
);

-- 3. Multi-Tenant Master Locker State Machine
CREATE TABLE IF NOT EXISTS locker_status (
    account_id INTEGER PRIMARY KEY, 
    latest_code TEXT NOT NULL,
    is_active INTEGER DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
);