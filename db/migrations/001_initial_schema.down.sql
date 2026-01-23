-- Drop views first
DROP VIEW IF EXISTS latest_test_results;
DROP VIEW IF EXISTS latest_test_executions;

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS test_executions;
DROP TABLE IF EXISTS collections;
