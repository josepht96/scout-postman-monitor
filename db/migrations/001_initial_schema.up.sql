-- Collections table: tracks all monitored Postman collections
CREATE TABLE IF NOT EXISTS collections (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Test executions table: stores each execution run of a collection
CREATE TABLE IF NOT EXISTS test_executions (
    id SERIAL PRIMARY KEY,
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    collection_name VARCHAR(255) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_ms INTEGER NOT NULL,
    total_tests INTEGER NOT NULL DEFAULT 0,
    passed_tests INTEGER NOT NULL DEFAULT 0,
    failed_tests INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster queries by collection
CREATE INDEX idx_test_executions_collection_id ON test_executions(collection_id);
CREATE INDEX idx_test_executions_started_at ON test_executions(started_at DESC);

-- Test results table: stores individual test results within each execution
CREATE TABLE IF NOT EXISTS test_results (
    id SERIAL PRIMARY KEY,
    execution_id INTEGER NOT NULL REFERENCES test_executions(id) ON DELETE CASCADE,
    test_name TEXT NOT NULL,
    execution_name VARCHAR(255),
    url TEXT,
    method VARCHAR(10),
    status VARCHAR(50) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    passed BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster queries
CREATE INDEX idx_test_results_execution_id ON test_results(execution_id);
CREATE INDEX idx_test_results_test_name ON test_results(test_name);

-- Latest results view: provides quick access to the most recent result for each collection
CREATE OR REPLACE VIEW latest_test_executions AS
SELECT DISTINCT ON (collection_id) *
FROM test_executions
ORDER BY collection_id, started_at DESC;

-- Latest test results view: provides the most recent result for each test
CREATE OR REPLACE VIEW latest_test_results AS
SELECT DISTINCT ON (tr.test_name, te.collection_id)
    tr.*,
    te.collection_id,
    te.collection_name,
    te.started_at as execution_started_at
FROM test_results tr
JOIN test_executions te ON tr.execution_id = te.id
ORDER BY tr.test_name, te.collection_id, te.started_at DESC;
