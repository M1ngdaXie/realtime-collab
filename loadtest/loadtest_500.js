import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// =============================================================================
// CUSTOM METRICS
// =============================================================================
const connectionTime = new Trend('ws_connection_time_ms');
const messageRTT = new Trend('ws_message_rtt_ms');
const connectionsFailed = new Counter('ws_connections_failed');
const messagesReceived = new Counter('ws_messages_received');
const messagesSent = new Counter('ws_messages_sent');
const connectionSuccess = new Rate('ws_connection_success');

// =============================================================================
// CONSERVATIVE TEST - 500 users (reliable on local machine)
// =============================================================================
export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '10s', target: 300 },
    { duration: '10s', target: 500 },
    { duration: '60s', target: 500 },  // Hold at 500 for 1 minute
    { duration: '10s', target: 0 },
  ],

  thresholds: {
    'ws_connection_time_ms': ['p(95)<500'],
    'ws_message_rtt_ms': ['p(95)<100'],
    'ws_connection_success': ['rate>0.99'],  // Expect 99%+ success at 500
  },
};

export default function () {
  const roomId = `loadtest-room-${__VU % 10}`;  // 10 rooms, 50 users each
  const url = `ws://localhost:8080/ws/loadtest/${roomId}`;

  const connectStart = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    const connectDuration = Date.now() - connectStart;
    connectionTime.add(connectDuration);
    connectionSuccess.add(1);

    let pingStart = 0;

    socket.on('message', (data) => {
      messagesReceived.add(1);
      if (pingStart > 0) {
        const rtt = Date.now() - pingStart;
        messageRTT.add(rtt);
        pingStart = 0;
      }
    });

    socket.on('error', (e) => {
      connectionsFailed.add(1);
    });

    // Send message every 500ms (more aggressive)
    socket.setInterval(function () {
      const payload = new Uint8Array([1, 0]);
      pingStart = Date.now();
      socket.sendBinary(payload.buffer);
      messagesSent.add(1);
    }, 500);

    socket.setTimeout(function () {
      socket.close();
    }, 90000);
  });

  const connected = check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  if (!connected) {
    connectionsFailed.add(1);
    connectionSuccess.add(0);
  }

  sleep(0.1);
}

export function setup() {
  console.log('========================================');
  console.log('  Conservative Load Test (500 users)');
  console.log('========================================');
  console.log('This test should show ~99% success rate');
  console.log('Use this for reliable Before/After comparison');
  console.log('========================================');
}
