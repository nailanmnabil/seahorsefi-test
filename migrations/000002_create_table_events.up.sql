CREATE TABLE
    IF NOT EXISTS events (
        id SERIAL PRIMARY KEY,
        address VARCHAR(255) NOT NULL,
        block_number BIGINT NOT NULL,
        transaction_hash VARCHAR(255) NOT NULL UNIQUE,
        event_type VARCHAR(255) NOT NULL,
        wallet_id VARCHAR(255) REFERENCES wallets (id) ON DELETE CASCADE ON UPDATE CASCADE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        last_calculated_at TIMESTAMP,
        closed_at TIMESTAMP
    );