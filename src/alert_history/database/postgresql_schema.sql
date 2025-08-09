-- PostgreSQL Schema для Alert History Service
-- Совместимая с существующей SQLite структурой для zero-downtime migration

-- Enable UUID extension for better performance
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable btree_gin for advanced indexing
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Alerts table - основная таблица для хранения алертов
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    fingerprint VARCHAR(64) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'firing',
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    starts_at TIMESTAMP WITH TIME ZONE,
    ends_at TIMESTAMP WITH TIME ZONE,
    generator_url TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- LLM Classifications table - для хранения результатов классификации
CREATE TABLE IF NOT EXISTS alert_classifications (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint VARCHAR(64) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    confidence DECIMAL(4,3) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    reasoning TEXT,
    recommendations JSONB DEFAULT '[]',
    processing_time DECIMAL(8,3),
    metadata JSONB DEFAULT '{}',
    llm_model VARCHAR(100),
    llm_version VARCHAR(50),
    cache_hit BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Publishing History table - для tracking публикаций в external systems
CREATE TABLE IF NOT EXISTS alert_publishing_history (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint VARCHAR(64) NOT NULL,
    target_name VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_format VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL, -- 'success', 'failure', 'filtered', 'retry'
    attempt_number INTEGER NOT NULL DEFAULT 1,
    response_code INTEGER,
    response_message TEXT,
    payload_size INTEGER,
    processing_time DECIMAL(8,3),
    error_details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Filter Rules table - для dynamic filter management
CREATE TABLE IF NOT EXISTS filter_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    target_name VARCHAR(100), -- NULL для global rules
    action VARCHAR(20) NOT NULL CHECK (action IN ('allow', 'deny', 'delay')),
    conditions JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Publishing Targets table - для хранения конфигурации targets
CREATE TABLE IF NOT EXISTS publishing_targets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    format VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    headers JSONB DEFAULT '{}',
    filter_config JSONB DEFAULT '{}',
    circuit_breaker_config JSONB DEFAULT '{}',
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    failure_count INTEGER DEFAULT 0,
    total_attempts INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- System Metrics table - для internal metrics tracking
CREATE TABLE IF NOT EXISTS system_metrics (
    id BIGSERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_type VARCHAR(20) NOT NULL, -- 'counter', 'gauge', 'histogram'
    labels JSONB DEFAULT '{}',
    value DECIMAL(15,6) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Migration History table - для tracking schema migrations
CREATE TABLE IF NOT EXISTS migration_history (
    id BIGSERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    execution_time DECIMAL(8,3),
    checksum VARCHAR(64)
);

-- ===============================
-- INDEXES для оптимальной производительности
-- ===============================

-- Alerts table indexes
CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint ON alerts(fingerprint);
CREATE INDEX IF NOT EXISTS idx_alerts_alert_name ON alerts(alert_name);
CREATE INDEX IF NOT EXISTS idx_alerts_namespace ON alerts(namespace);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_timestamp ON alerts(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_starts_at ON alerts(starts_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);

-- GIN indexes для JSONB поиска
CREATE INDEX IF NOT EXISTS idx_alerts_labels_gin ON alerts USING GIN(labels);
CREATE INDEX IF NOT EXISTS idx_alerts_annotations_gin ON alerts USING GIN(annotations);

-- Composite indexes для частых запросов
CREATE INDEX IF NOT EXISTS idx_alerts_name_status_time ON alerts(alert_name, status, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint_status ON alerts(fingerprint, status);

-- Alert Classifications indexes
CREATE INDEX IF NOT EXISTS idx_classifications_fingerprint ON alert_classifications(alert_fingerprint);
CREATE INDEX IF NOT EXISTS idx_classifications_severity ON alert_classifications(severity);
CREATE INDEX IF NOT EXISTS idx_classifications_confidence ON alert_classifications(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_classifications_created_at ON alert_classifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_classifications_expires_at ON alert_classifications(expires_at) WHERE expires_at IS NOT NULL;

-- Publishing History indexes
CREATE INDEX IF NOT EXISTS idx_publishing_history_fingerprint ON alert_publishing_history(alert_fingerprint);
CREATE INDEX IF NOT EXISTS idx_publishing_history_target ON alert_publishing_history(target_name);
CREATE INDEX IF NOT EXISTS idx_publishing_history_status ON alert_publishing_history(status);
CREATE INDEX IF NOT EXISTS idx_publishing_history_created_at ON alert_publishing_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_publishing_history_target_status_time ON alert_publishing_history(target_name, status, created_at DESC);

-- Filter Rules indexes
CREATE INDEX IF NOT EXISTS idx_filter_rules_target ON filter_rules(target_name) WHERE target_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_filter_rules_priority ON filter_rules(priority) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_filter_rules_enabled ON filter_rules(enabled, priority);

-- Publishing Targets indexes
CREATE INDEX IF NOT EXISTS idx_publishing_targets_enabled ON publishing_targets(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_publishing_targets_type ON publishing_targets(type);

-- System Metrics indexes
CREATE INDEX IF NOT EXISTS idx_system_metrics_name_time ON system_metrics(metric_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_metrics_timestamp ON system_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_metrics_labels_gin ON system_metrics USING GIN(labels);

-- ===============================
-- TRIGGERS для автоматического обновления updated_at
-- ===============================

-- Function для обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers
CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_classifications_updated_at
    BEFORE UPDATE ON alert_classifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_filter_rules_updated_at
    BEFORE UPDATE ON filter_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_publishing_targets_updated_at
    BEFORE UPDATE ON publishing_targets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ===============================
-- VIEWS для удобного доступа к данным
-- ===============================

-- View для combined alert data с классификацией
CREATE OR REPLACE VIEW alerts_with_classification AS
SELECT
    a.*,
    ac.severity as llm_severity,
    ac.confidence as llm_confidence,
    ac.reasoning as llm_reasoning,
    ac.recommendations as llm_recommendations,
    ac.llm_model,
    ac.created_at as classification_time
FROM alerts a
LEFT JOIN alert_classifications ac ON a.fingerprint = ac.alert_fingerprint
    AND ac.created_at = (
        SELECT MAX(created_at)
        FROM alert_classifications ac2
        WHERE ac2.alert_fingerprint = a.fingerprint
    );

-- View для publishing statistics
CREATE OR REPLACE VIEW publishing_stats AS
SELECT
    target_name,
    target_type,
    target_format,
    COUNT(*) as total_attempts,
    COUNT(*) FILTER (WHERE status = 'success') as successful_publishes,
    COUNT(*) FILTER (WHERE status = 'failure') as failed_publishes,
    COUNT(*) FILTER (WHERE status = 'filtered') as filtered_publishes,
    AVG(processing_time) FILTER (WHERE status = 'success') as avg_success_time,
    MAX(created_at) as last_publish_time
FROM alert_publishing_history
GROUP BY target_name, target_type, target_format;

-- View для recent alerts (последние 24 часа)
CREATE OR REPLACE VIEW recent_alerts AS
SELECT * FROM alerts_with_classification
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- ===============================
-- PARTITIONING для больших объемов данных
-- ===============================

-- Партиционирование для alert_publishing_history по времени (месячные партиции)
-- Это поможет при больших объемах data

-- Функция для автоматического создания партиций
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name text, start_date date)
RETURNS void AS $$
DECLARE
    partition_name text;
    end_date date;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + interval '1 month';

    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
                    FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- ===============================
-- DATA RETENTION политики
-- ===============================

-- Function для cleanup старых данных
CREATE OR REPLACE FUNCTION cleanup_old_data(retention_days integer DEFAULT 30)
RETURNS integer AS $$
DECLARE
    deleted_count integer := 0;
    cutoff_date timestamp with time zone;
BEGIN
    cutoff_date := NOW() - (retention_days || ' days')::interval;

    -- Clean up old alerts
    DELETE FROM alerts WHERE created_at < cutoff_date;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    -- Clean up old publishing history (keep longer for analytics)
    DELETE FROM alert_publishing_history WHERE created_at < cutoff_date - interval '30 days';

    -- Clean up expired classifications
    DELETE FROM alert_classifications WHERE expires_at IS NOT NULL AND expires_at < NOW();

    -- Clean up old system metrics (keep last 7 days)
    DELETE FROM system_metrics WHERE timestamp < NOW() - interval '7 days';

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ===============================
-- INITIAL DATA
-- ===============================

-- Insert default filter rules
INSERT INTO filter_rules (name, action, conditions, priority, enabled) VALUES
('block_noise_alerts', 'deny', '{"llm_severity": "noise"}', 10, TRUE),
('allow_critical_alerts', 'allow', '{"llm_severity": "critical"}', 1, TRUE),
('min_confidence_threshold', 'deny', '{"llm_confidence_below": 0.3, "has_llm_classification": true}', 20, TRUE)
ON CONFLICT (name) DO NOTHING;

-- Insert initial migration record
INSERT INTO migration_history (version, description) VALUES
('001_initial_schema', 'Initial PostgreSQL schema creation')
ON CONFLICT (version) DO NOTHING;

-- ===============================
-- PERFORMANCE TUNING
-- ===============================

-- Настройки для оптимальной производительности
-- (выполняются администратором базы данных)

-- VACUUM и ANALYZE настройки
ALTER TABLE alerts SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE alert_classifications SET (autovacuum_vacuum_scale_factor = 0.2);
ALTER TABLE alert_publishing_history SET (autovacuum_vacuum_scale_factor = 0.2);

-- Статистика для оптимизатора
ALTER TABLE alerts ALTER COLUMN labels SET STATISTICS 1000;
ALTER TABLE alerts ALTER COLUMN annotations SET STATISTICS 1000;
ALTER TABLE alert_classifications ALTER COLUMN metadata SET STATISTICS 500;
