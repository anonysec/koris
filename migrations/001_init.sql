-- KorisPanel Lite — Initial Schema
-- Supports: Settings, Users, Nodes, OpenVPN, L2TP

-- Admin accounts
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('owner','admin') NOT NULL DEFAULT 'admin',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Admin sessions
CREATE TABLE IF NOT EXISTS admin_sessions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    username VARCHAR(64) NOT NULL,
    role VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB;

-- Panel settings (key-value)
CREATE TABLE IF NOT EXISTS panel_settings (
    key_name VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- VPN Customers
CREATE TABLE IF NOT EXISTS customers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL COMMENT 'Cleartext for RADIUS CHAP',
    portal_password_hash VARCHAR(255),
    status ENUM('active','disabled','expired','deleted') DEFAULT 'active',
    data_limit_gb DECIMAL(12,2) DEFAULT 50,
    data_used_gb DECIMAL(12,2) DEFAULT 0,
    max_sessions INT DEFAULT 1,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_username (username)
) ENGINE=InnoDB;

-- VPN Nodes
CREATE TABLE IF NOT EXISTS nodes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    public_ip VARCHAR(45) NOT NULL,
    api_token_hash VARCHAR(64) NOT NULL,
    status ENUM('online','offline','stale') DEFAULT 'offline',
    openvpn_enabled BOOLEAN DEFAULT TRUE,
    l2tp_enabled BOOLEAN DEFAULT TRUE,
    last_seen_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status)
) ENGINE=InnoDB;

-- Node tasks (dispatched from panel to node agent)
CREATE TABLE IF NOT EXISTS node_tasks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL,
    payload_json JSON,
    status ENUM('pending','running','completed','failed') DEFAULT 'pending',
    result_json JSON,
    error TEXT,
    created_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    INDEX idx_node_status (node_id, status)
) ENGINE=InnoDB;

-- FreeRADIUS tables (standard schema)
CREATE TABLE IF NOT EXISTS radcheck (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    attribute VARCHAR(64) NOT NULL,
    op CHAR(2) NOT NULL DEFAULT ':=',
    value VARCHAR(253) NOT NULL,
    INDEX idx_username (username)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS radreply (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    attribute VARCHAR(64) NOT NULL,
    op CHAR(2) NOT NULL DEFAULT ':=',
    value VARCHAR(253) NOT NULL,
    INDEX idx_username (username)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS radacct (
    radacctid BIGINT AUTO_INCREMENT PRIMARY KEY,
    acctsessionid VARCHAR(64) NOT NULL DEFAULT '',
    acctuniqueid VARCHAR(32) NOT NULL DEFAULT '',
    username VARCHAR(64) NOT NULL DEFAULT '',
    nasipaddress VARCHAR(15) NOT NULL DEFAULT '',
    nasportid VARCHAR(32),
    nasporttype VARCHAR(32),
    acctstarttime DATETIME,
    acctupdatetime DATETIME,
    acctstoptime DATETIME,
    acctinputoctets BIGINT DEFAULT 0,
    acctoutputoctets BIGINT DEFAULT 0,
    calledstationid VARCHAR(50) DEFAULT '',
    callingstationid VARCHAR(50) DEFAULT '',
    acctterminatecause VARCHAR(32) DEFAULT '',
    framedipaddress VARCHAR(15) DEFAULT '',
    INDEX idx_username (username),
    INDEX idx_start (acctstarttime),
    INDEX idx_stop (acctstoptime)
) ENGINE=InnoDB;

-- Default settings
INSERT INTO panel_settings (key_name, value) VALUES
('panel_name', 'KorisPanel Lite'),
('language', 'en')
ON DUPLICATE KEY UPDATE value=value;
