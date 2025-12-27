----------------------- MySQL -----------------------
CREATE TABLE IF NOT EXISTS distributed_lock (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    lock_key VARCHAR(128) NOT NULL,
    lock_value VARCHAR(36) NOT NULL,
    expire_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY (lock_key),
    INDEX (expire_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
------------------------------------------------------

--------------------- PostgreSQL ---------------------
CREATE TABLE IF NOT EXISTS distributed_lock (
    id SERIAL PRIMARY KEY,
    lock_key VARCHAR(128) NOT NULL UNIQUE,
    lock_value VARCHAR(36) NOT NULL,
    expire_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_expire_time ON distributed_lock (expire_time);
------------------------------------------------------

----------------------- SQLite -----------------------
CREATE TABLE IF NOT EXISTS distributed_lock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lock_key TEXT NOT NULL UNIQUE,
    lock_value TEXT NOT NULL,
    expire_time DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_expire_time ON distributed_lock (expire_time);
------------------------------------------------------