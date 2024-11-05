CREATE TABLE
    IF NOT EXISTS wallets (
        id VARCHAR(255) PRIMARY KEY,
        address VARCHAR(255) UNIQUE NOT NULL,
        points BIGINT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP
    );