import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// =============================================================================
// PRODUCTION-STYLE LOAD TEST (CONFIGURABLE)
// =============================================================================
// Usage examples:
//   k6 run loadtest/loadtest_prod.js
//   SCENARIOS=connect k6 run loadtest/loadtest_prod.js
//   SCENARIOS=connect,fanout ROOMS=10 FANOUT_VUS=1000 k6 run loadtest/loadtest_prod.js
//   BASE_URL=ws://localhost:8080/ws/loadtest k6 run loadtest/loadtest_prod.js
//
// Notes:
// - Default uses /ws/loadtest to bypass auth + DB permission checks.
// - RTT is not measured (server does not echo to sender).
// - Use SCENARIOS to isolate connection-only vs fanout pressure.

const BASE_URL = __ENV.BASE_URL || 'ws://localhost:8080/ws/loadtest';
const ROOMS = parseInt(__ENV.ROOMS || '10', 10);
const SEND_INTERVAL_MS = parseInt(__ENV.SEND_INTERVAL_MS || '500', 10);
const PAYLOAD_BYTES = parseInt(__ENV.PAYLOAD_BYTES || '200', 10);
const CONNECT_HOLD_SEC = parseInt(__ENV.CONNECT_HOLD_SEC || '30', 10);
const SCENARIOS = (__ENV.SCENARIOS || 'connect,fanout').split(',').map((s) => s.trim());

// =============================================================================
// CUSTOM METRICS
// =============================================================================
const connectionTime = new Trend('ws_connection_time_ms');
const connectionsFailed = new Counter('ws_connections_failed');
const messagesReceived = new Counter('ws_msgs_received');
const messagesSent = new Counter('ws_msgs_sent');
const connectionSuccess = new Rate('ws_connection_success');

function roomForVU() {
  return `loadtest-room-${__VU % ROOMS}`;
}

function buildUrl(roomId) {
  return `${BASE_URL}/${roomId}`;
}

function connectAndHold(roomId, holdSec) {
  const url = buildUrl(roomId);
  const connectStart = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    connectionTime.add(Date.now() - connectStart);
    connectionSuccess.add(1);

    socket.on('message', () => {
      messagesReceived.add(1);
    });

    socket.on('error', () => {
      connectionsFailed.add(1);
    });

    socket.setTimeout(() => {
      socket.close();
    }, holdSec * 1000);
  });

  const connected = check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  if (!connected) {
    connectionsFailed.add(1);
    connectionSuccess.add(0);
  }
}

function connectAndFanout(roomId) {
  const url = buildUrl(roomId);
  const connectStart = Date.now();
  const payload = new Uint8Array(PAYLOAD_BYTES);
  payload[0] = 1;
  for (let i = 1; i < PAYLOAD_BYTES; i++) {
    payload[i] = Math.floor(Math.random() * 256);
  }

  const res = ws.connect(url, {}, function (socket) {
    connectionTime.add(Date.now() - connectStart);
    connectionSuccess.add(1);

    socket.on('message', () => {
      messagesReceived.add(1);
    });

    socket.on('error', () => {
      connectionsFailed.add(1);
    });

    socket.setInterval(() => {
      socket.sendBinary(payload.buffer);
      messagesSent.add(1);
    }, SEND_INTERVAL_MS);

    socket.setTimeout(() => {
      socket.close();
    }, CONNECT_HOLD_SEC * 1000);
  });

  const connected = check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  if (!connected) {
    connectionsFailed.add(1);
    connectionSuccess.add(0);
  }
}

// =============================================================================
// SCENARIOS (decided at init time from env)
// =============================================================================
const scenarios = {};

if (SCENARIOS.includes('connect')) {
  scenarios.connect_only = {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 200 },
      { duration: '10s', target: 500 },
      { duration: '10s', target: 1000 },
      { duration: '60s', target: 1000 },
      { duration: '10s', target: 0 },
    ],
    exec: 'connectOnly',
  };
}

if (SCENARIOS.includes('fanout')) {
  scenarios.fanout = {
    executor: 'constant-vus',
    vus: parseInt(__ENV.FANOUT_VUS || '1000', 10),
    duration: __ENV.FANOUT_DURATION || '90s',
    exec: 'fanout',
  };
}

export const options = {
  scenarios,
  thresholds: {
    ws_connection_time_ms: ['p(95)<500'],
    ws_connection_success: ['rate>0.95'],
  },
};

export function connectOnly() {
  connectAndHold(roomForVU(), CONNECT_HOLD_SEC);
  sleep(0.1);
}

export function fanout() {
  connectAndFanout(roomForVU());
  sleep(0.1);
}

export function setup() {
  console.log('========================================');
  console.log(' Production-Style Load Test');
  console.log('========================================');
  console.log(`BASE_URL: ${BASE_URL}`);
  console.log(`ROOMS: ${ROOMS}`);
  console.log(`SCENARIOS: ${SCENARIOS.join(',')}`);
  console.log(`SEND_INTERVAL_MS: ${SEND_INTERVAL_MS}`);
  console.log(`PAYLOAD_BYTES: ${PAYLOAD_BYTES}`);
  console.log(`CONNECT_HOLD_SEC: ${CONNECT_HOLD_SEC}`);
  console.log('========================================');
}
