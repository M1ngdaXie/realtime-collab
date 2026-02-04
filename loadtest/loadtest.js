import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// =============================================================================
// CUSTOM METRICS - These will show up in your test results
// =============================================================================
const connectionTime = new Trend('ws_connection_time_ms');      // Connection latency
const messageRTT = new Trend('ws_message_rtt_ms');              // Round-trip time
const connectionsFailed = new Counter('ws_connections_failed'); // Failed connections
const messagesReceived = new Counter('ws_messages_received');   // Messages received
const messagesSent = new Counter('ws_messages_sent');           // Messages sent
const connectionSuccess = new Rate('ws_connection_success');    // Success rate

// =============================================================================
// TEST CONFIGURATION
// =============================================================================
export const options = {
  stages: [
    // Ramp up phase: 0 → 2000 connections over 30 seconds
    { duration: '10s', target: 500 },
    { duration: '10s', target: 1000 },
    { duration: '10s', target: 2000 },

    // Hold at peak: maintain 2000 connections for 1 minute
    { duration: '60s', target: 2000 },

    // Ramp down: graceful disconnect
    { duration: '10s', target: 0 },
  ],

  thresholds: {
    'ws_connection_time_ms': ['p(95)<1000'],    // 95% connect under 1s
    'ws_message_rtt_ms': ['p(95)<100'],          // 95% RTT under 100ms
    'ws_connection_success': ['rate>0.95'],      // 95% success rate
  },
};

// =============================================================================
// MAIN TEST FUNCTION - Runs for each virtual user (VU)
// =============================================================================
export default function () {
  // Distribute users across 20 rooms (simulates realistic usage)
  const roomId = `loadtest-room-${__VU % 20}`;
  const url = `ws://localhost:8080/ws/loadtest/${roomId}`;

  const connectStart = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    // Track connection time
    const connectDuration = Date.now() - connectStart;
    connectionTime.add(connectDuration);
    connectionSuccess.add(1);

    // Variables for RTT measurement
    let pendingPing = null;
    let pingStart = 0;

    // =======================================================================
    // MESSAGE HANDLER - Receives broadcasts from other users
    // =======================================================================
    socket.on('message', (data) => {
      messagesReceived.add(1);

      // Check if this is our ping response (echo from server)
      if (pendingPing && data.length > 0) {
        const rtt = Date.now() - pingStart;
        messageRTT.add(rtt);
        pendingPing = null;
      }
    });

    // =======================================================================
    // ERROR HANDLER
    // =======================================================================
    socket.on('error', (e) => {
      console.error(`WebSocket error: ${e.error()}`);
      connectionsFailed.add(1);
    });

    // =======================================================================
    // PING LOOP - Send a message every second to measure RTT
    // =======================================================================
    socket.setInterval(function () {
      // Create a simple binary message (mimics Yjs awareness update)
      // Format: [1, ...payload] where 1 = Awareness message type
      const timestamp = Date.now();
      const payload = new Uint8Array([
        1,  // Message type: Awareness
        0,  // Payload length (varint)
      ]);

      pingStart = timestamp;
      pendingPing = true;

      socket.sendBinary(payload.buffer);
      messagesSent.add(1);

    }, 1000); // Every 1 second

    // =======================================================================
    // STAY CONNECTED - Keep connection alive for the test duration
    // =======================================================================
    socket.setTimeout(function () {
      socket.close();
    }, 90000); // 90 seconds max per connection
  });

  // =======================================================================
  // CONNECTION RESULT CHECK
  // =======================================================================
  const connected = check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  if (!connected) {
    connectionsFailed.add(1);
    connectionSuccess.add(0);
  }

  // Small sleep to prevent tight loop on connection failure
  sleep(0.1);
}

// =============================================================================
// LIFECYCLE HOOKS
// =============================================================================
export function setup() {
  console.log('========================================');
  console.log('    WebSocket Load Test Starting');
  console.log('========================================');
  console.log('Target: ws://localhost:8080/ws/loadtest/:roomId');
  console.log('Max VUs: 2000');
  console.log('Duration: ~100 seconds');
  console.log('');
  console.log('Key Metrics to Watch:');
  console.log('  - ws_connection_time_ms (p95 < 1s)');
  console.log('  - ws_message_rtt_ms (p95 < 100ms)');
  console.log('  - ws_connection_success (rate > 95%)');
  console.log('========================================');
}

export function teardown(data) {
  console.log('========================================');
  console.log('    Load Test Complete!');
  console.log('========================================');
  console.log('');
  console.log('📸 Take a screenshot of the results above');
  console.log('   for your "Before vs After" comparison!');
  console.log('========================================');
}
