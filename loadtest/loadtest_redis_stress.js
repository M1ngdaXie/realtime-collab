import { check, sleep } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import ws from "k6/ws";

// =============================================================================
// CUSTOM METRICS (avoid conflicts with k6's built-in ws_* metrics)
// =============================================================================
const connectionTime = new Trend("ws_connection_time_ms");
const messageRTT = new Trend("ws_message_rtt_ms");
const connectionsFailed = new Counter("ws_connections_failed");
const messagesReceived = new Counter("ws_msgs_received");
const messagesSent = new Counter("ws_msgs_sent");
const connectionSuccess = new Rate("ws_connection_success");

// =============================================================================
// 1000 USERS TEST - STRESS REDIS PUBSUB SUBSCRIPTIONS
// =============================================================================
export const options = {
  stages: [
    { duration: "20s", target: 20 }, // Warmup: 20 users
    { duration: "10s", target: 200 }, // Ramp to 200
    { duration: "10s", target: 500 }, // Ramp to 500
    { duration: "10s", target: 1000 }, // Ramp to 1000
    { duration: "60s", target: 1000 }, // Hold at 1000 for 1 minute
    { duration: "10s", target: 0 }, // Ramp down
  ],

  thresholds: {
    ws_connection_time_ms: ["p(95)<500"], // Target: <500ms connection
    ws_message_rtt_ms: ["p(95)<100"], // Target: <100ms message RTT
    ws_connection_success: ["rate>0.95"], // Target: >95% success rate
  },
};

export default function () {
  // CRITICAL: Create unique room per user to stress Redis PubSub
  // This creates ~1000 subscriptions (1 per room) to trigger the bottleneck
  const roomId = `loadtest-room-${__VU}`;
  const url = `ws://localhost:8080/ws/loadtest/${roomId}`;

  const connectStart = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    const connectDuration = Date.now() - connectStart;
    connectionTime.add(connectDuration);
    connectionSuccess.add(1);

    // Send realistic Yjs-sized messages (200 bytes)
    // Yjs sync messages are typically 100-500 bytes
    const payload = new Uint8Array(200);
    payload[0] = 1; // Message type: Yjs sync

    // Fill with realistic data (not zeros)
    for (let i = 1; i < 200; i++) {
      payload[i] = Math.floor(Math.random() * 256);
    }

    socket.on("message", (data) => {
      messagesReceived.add(1);
    });

    socket.on("error", (e) => {
      connectionsFailed.add(1);
    });

    // Send message every 1 second (realistic collaborative edit rate)
    socket.setInterval(function () {
      socket.sendBinary(payload.buffer);
      messagesSent.add(1);
    }, 1000);

    // Keep connection alive for 100 seconds (longer than test duration)
    socket.setTimeout(function () {
      socket.close();
    }, 100000);
  });

  const connectCheck = check(res, {
    "WebSocket connected": (r) => r && r.status === 101,
  });

  if (!connectCheck) {
    connectionsFailed.add(1);
    connectionSuccess.add(0);
  }

  // Small sleep to avoid hammering connection endpoint
  sleep(0.1);
}

export function setup() {
  console.log("========================================");
  console.log("  Redis PubSub Stress Test: 1000 Users");
  console.log("========================================");
  console.log("⚠️  CRITICAL: Creates ~1000 rooms");
  console.log("   This stresses Redis PubSub subscriptions");
  console.log("   Each room = 1 dedicated PubSub connection");
  console.log("========================================");
  console.log("Expected bottleneck (before fix):");
  console.log("  - 58.96s in PubSub health checks");
  console.log("  - 28.09s in ReceiveTimeout");
  console.log("  - Connection success rate: 80-85%");
  console.log("  - P95 latency: 20-26 seconds");
  console.log("========================================");
  console.log("Expected after fix:");
  console.log("  - <1s in PubSub operations");
  console.log("  - Connection success rate: >95%");
  console.log("  - P95 latency: <500ms");
  console.log("========================================");
}

export function teardown(data) {
  console.log("========================================");
  console.log("  Load Test Completed");
  console.log("========================================");
  console.log("Check profiling data with:");
  console.log("  curl http://localhost:8080/debug/pprof/mutex > mutex.pb");
  console.log("  go tool pprof -top mutex.pb");
  console.log("========================================");
}
