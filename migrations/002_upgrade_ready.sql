-- Migration 002: Prepare schema for upgrade to full KorisPanel
-- Adds columns/tables that the main panel expects, so migration is seamless

-- Add columns the full panel uses
ALTER TABLE customers ADD COLUMN IF NOT EXISTS plan_id BIGINT NULL;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS wallet_balance DECIMAL(12,2) DEFAULT 0;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS auto_renew BOOLEAN DEFAULT FALSE;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS node_id BIGINT NULL;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS created_by VARCHAR(64) DEFAULT 'admin';

ALTER TABLE nodes ADD COLUMN IF NOT EXISTS domain VARCHAR(255) DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS ikev2_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS wireguard_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS ssh_enabled BOOLEAN DEFAULT FALSE;

-- Audit log (used by full panel)
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    actor VARCHAR(64) NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(50),
    before_json JSON,
    after_json JSON,
    ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_actor (actor),
    INDEX idx_created (created_at)
) ENGINE=InnoDB;

-- Events table (used by full panel for notifications)
CREATE TABLE IF NOT EXISTS events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    severity ENUM('info','warning','error','critical') DEFAULT 'info',
    title VARCHAR(255) NOT NULL,
    message TEXT,
    actor VARCHAR(64),
    related VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type (type),
    INDEX idx_created (created_at)
) ENGINE=InnoDB;
