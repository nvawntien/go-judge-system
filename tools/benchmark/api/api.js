// AstraCode read-only API capacity workload. It deliberately issues GET only.
import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL;
const endpoint = __ENV.API_PATH;
const outputDir = __ENV.OUTPUT_DIR;
const rate = Number(__ENV.TARGET_RPS);
const duration = __ENV.DURATION;
const systemConfig = loadSystemConfig(__ENV.SYSTEM_CONFIG_PATH);

export const options = {
  scenarios: {
    public_read: {
      executor: 'constant-arrival-rate',
      rate,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Number(__ENV.PREALLOCATED_VUS),
      maxVUs: Number(__ENV.MAX_VUS),
    },
  },
  thresholds: {},
};

export default function () {
  const response = http.get(`${baseURL}${endpoint}`, { tags: { endpoint } });
  check(response, { 'public problem response is successful': (value) => value.status >= 200 && value.status < 300 });
}

function metric(data, name, field, fallback = null) {
  const value = data.metrics[name] && data.metrics[name].values;
  return value && value[field] !== undefined ? value[field] : fallback;
}

export function handleSummary(data) {
  const started = new Date().toISOString();
  const run = {
    schema_version: 'astracode.api-benchmark.run.v1', benchmark_type: 'api', run_id: __ENV.RUN_ID,
    started_at: __ENV.STARTED_AT, ended_at: started,
    repository: { git_sha: __ENV.GIT_SHA, dirty: __ENV.GIT_DIRTY === 'true' },
    target: { base_url: baseURL, endpoint },
    workload: { requested_rps: rate, duration, preallocated_vus: Number(__ENV.PREALLOCATED_VUS), max_vus: Number(__ENV.MAX_VUS), max_requests: Number(__ENV.MAX_REQUESTS) },
    system_config: systemConfig,
  };
  const summary = {
    schema_version: 'astracode.api-benchmark.summary.v1', benchmark_type: 'api', run_id: __ENV.RUN_ID,
    requested_rps: rate, achieved_rps: metric(data, 'http_reqs', 'rate'), total_requests: metric(data, 'http_reqs', 'count', 0),
    successful_requests: metric(data, 'checks', 'passes', 0), failed_requests: metric(data, 'checks', 'fails', 0),
    error_rate: metric(data, 'http_req_failed', 'rate'), dropped_iterations: metric(data, 'dropped_iterations', 'count', 0),
    latency_ms: { p50: metric(data, 'http_req_duration', 'med'), p90: metric(data, 'http_req_duration', 'p(90)'), p95: metric(data, 'http_req_duration', 'p(95)'), p99: metric(data, 'http_req_duration', 'p(99)'), max: metric(data, 'http_req_duration', 'max') },
    data_received_per_second: metric(data, 'data_received', 'rate'),
  };
  return { [`${outputDir}/run.json`]: JSON.stringify(run, null, 2), [`${outputDir}/summary.json`]: JSON.stringify(summary, null, 2) };
}

function loadSystemConfig(path) {
  if (!path) return null;
  const value = JSON.parse(open(path));
  const top = ['label', 'release', 'app', 'judge'];
  if (!exactKeys(value, top) || !exactKeys(value.app, ['nodes', 'cpu_cores_per_node', 'memory_mib_per_node']) || !exactKeys(value.judge, ['nodes', 'cpu_cores_per_node', 'memory_mib_per_node', 'worker_pool_size', 'worker_memory_limit_mib', 'sandbox_memory_limit_mib'])) throw new Error('invalid strict system config');
  if (typeof value.label !== 'string' || !value.label || typeof value.release !== 'string' || !value.release) throw new Error('invalid system config label/release');
  for (const section of [value.app, value.judge]) for (const key in section) if (!Number.isInteger(section[key]) || section[key] <= 0) throw new Error('invalid positive system config value');
  return value;
}

function exactKeys(value, keys) {
  return value && typeof value === 'object' && Object.keys(value).length === keys.length && keys.every((key) => Object.prototype.hasOwnProperty.call(value, key));
}
