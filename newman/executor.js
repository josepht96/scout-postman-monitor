#!/usr/bin/env node

const newman = require('newman');
const path = require('path');

// Get collection path and optional environment path from command line arguments
const collectionPath = process.argv[2];
const environmentPath = process.argv[3]; // Optional
const directoryName = process.argv[4]; // Optional - directory name for secret injection
const environmentName = process.argv[5]; // Optional - environment name for secret injection

// Log all non-null arguments
console.error('[INFO] Executor arguments:');
if (collectionPath) console.error(`[INFO]   collectionPath: ${collectionPath}`);
if (environmentPath) console.error(`[INFO]   environmentPath: ${environmentPath}`);
if (directoryName) console.error(`[INFO]   directoryName: ${directoryName}`);
if (environmentName) console.error(`[INFO]   environmentName: ${environmentName}`);

if (!collectionPath) {
  console.error(JSON.stringify({
    error: 'Collection path is required',
    usage: 'node executor.js <collection-path> [environment-path]'
  }));
  process.exit(1);
}

// Load the collection to get its name
let collectionData;
try {
  collectionData = require(path.resolve(collectionPath));
} catch (e) {
  console.error(JSON.stringify({
    error: 'Failed to load collection: ' + e.message,
    collectionPath: collectionPath
  }));
  process.exit(1);
}

// Extract collection name from various possible locations
const collectionName = collectionData.info?.name ||
                        collectionData.name ||
                        path.basename(collectionPath, '.json');

// Load environment file if provided
let environmentData = null;
if (environmentPath) {
  try {
    environmentData = require(path.resolve(environmentPath));
  } catch (e) {
    console.error(JSON.stringify({
      error: 'Failed to load environment: ' + e.message,
      environmentPath: environmentPath
    }));
    process.exit(1);
  }
}

// Scan for secret environment variables to inject
const envVars = [];
if (directoryName && environmentName) {
  const prefix = `${directoryName}_${environmentName}_`;

  for (const key in process.env) {
    if (key.startsWith(prefix)) {
      const strippedKey = key.substring(prefix.length);
      envVars.push({
        key: strippedKey,
        value: process.env[key]
      });
      console.error(`[INFO] Injecting secret: ${strippedKey} from env var: ${key}`);
    }
  }

  if (envVars.length > 0) {
    console.error(`[INFO] Injected ${envVars.length} secret(s) from environment variables`);
  }
}

// Prepare result object
const result = {
  collectionName: collectionName,
  collectionPath: collectionPath,
  timestamp: new Date().toISOString(),
  summary: {
    total: 0,
    passed: 0,
    failed: 0
  },
  tests: [],
  executions: [],
  totalDurationMs: 0,
  error: null
};

// Run Newman
const runOptions = {
  collection: collectionData,
  reporters: [] // We'll handle reporting ourselves
};

// Add environment if provided
if (environmentData) {
  runOptions.environment = environmentData;
}

// Add injected environment variables if any
if (envVars.length > 0) {
  runOptions.envVar = envVars;
}

// Log the equivalent Newman CLI command for debugging
let cliCommand = `newman run ${collectionPath}`;
if (environmentPath) {
  cliCommand += ` --environment ${environmentPath}`;
}
if (envVars.length > 0) {
  envVars.forEach(envVar => {
    cliCommand += ` --env-var "${envVar.key}=${envVar.value}"`;
  });
}
console.error(`[INFO] Executing Newman command:\n${cliCommand}`);

newman.run(runOptions, (err) => {
  if (err) {
    result.error = err.message;
    console.log(JSON.stringify(result, null, 2));
    process.exit(1);
  }
}).on('start', (err, args) => {
  if (err) {
    result.error = err.message;
    return;
  }
  // Update collection name if available from args
  if (args && args.cursor && args.cursor.collection && args.cursor.collection.name) {
    result.collectionName = args.cursor.collection.name;
  }
}).on('request', (err, args) => {
  if (!args) return;

  const execution = {
    name: args.item?.name || 'Unknown Request',
    url: args.request?.url?.toString() || '',
    method: args.request?.method || 'GET',
    status: 'unknown',
    statusCode: null,
    responseTime: null,
    error: null
  };

  if (err) {
    execution.status = 'error';
    execution.error = err.message;
  } else if (args.response) {
    execution.statusCode = args.response.code;
    execution.responseTime = args.response.responseTime;
    execution.status = args.response.code >= 200 && args.response.code < 300 ? 'success' : 'failed';
  }

  result.executions.push(execution);
}).on('assertion', (err, args) => {
  if (!args) return;

  const test = {
    name: args.assertion || 'Unknown Test',
    passed: !err,
    error: err ? err.message : null,
    executionName: args.item?.name || 'unknown'
  };

  result.tests.push(test);
  result.summary.total++;

  if (test.passed) {
    result.summary.passed++;
  } else {
    result.summary.failed++;
  }
}).on('done', (err, summary) => {
  if (err) {
    result.error = err.message;
  }

  if (summary) {
    result.totalDurationMs = summary.run.timings.completed - summary.run.timings.started;
  }

  // Output the final result as JSON
  console.log(JSON.stringify(result, null, 2));
});
