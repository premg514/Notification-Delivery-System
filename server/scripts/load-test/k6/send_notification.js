import http from "k6/http";
import { check } from "k6";
import { Counter, Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const SCENARIO = (__ENV.SCENARIO || "baseline").toLowerCase();
const TEST_RUN_ID = __ENV.TEST_RUN_ID || "local";
const MAX_RESPONSE_MS = Number(__ENV.MAX_RESPONSE_MS || 1000);
const MAX_ERROR_RATE = Number(__ENV.MAX_ERROR_RATE || 0.02);
const DEPARTMENTS = ["CSE", "ECE", "ME", "CIVIL", "EEE"];

const scenarios = {
  smoke: {
    executor: "constant-vus",
    vus: 5,
    duration: "1m",
  },
  baseline: {
    executor: "ramping-vus",
    stages: [
      { duration: "2m", target: 50 },
      { duration: "5m", target: 50 },
      { duration: "2m", target: 0 },
    ],
    gracefulRampDown: "30s",
  },
  spike: {
    executor: "ramping-vus",
    stages: [
      { duration: "1m", target: 50 },
      { duration: "2m", target: 400 },
      { duration: "2m", target: 400 },
      { duration: "1m", target: 50 },
      { duration: "1m", target: 0 },
    ],
    gracefulRampDown: "30s",
  },
  soak: {
    executor: "constant-vus",
    vus: 120,
    duration: "30m",
  },
};

const chosenScenario = scenarios[SCENARIO] || scenarios.baseline;

const sendErrors = new Counter("send_notification_errors_total");
const acceptedCount = new Counter("send_notification_accepted_total");
const requestDuration = new Trend("send_notification_duration_ms", true);

export const options = {
  scenarios: {
    send_notifications: chosenScenario,
  },
  thresholds: {
    http_req_failed: [`rate<${MAX_ERROR_RATE}`],
    http_req_duration: [`p(95)<${MAX_RESPONSE_MS}`],
    send_notification_duration_ms: [`p(95)<${MAX_RESPONSE_MS}`],
  },
  summaryTrendStats: ["avg", "min", "med", "p(90)", "p(95)", "max"],
};

function buildPayload(iteration) {
  return {
    title: `Load Test Notification ${iteration}`,
    message: "Training session starts tomorrow at 9 AM.",
    target_department: DEPARTMENTS[iteration % DEPARTMENTS.length],
    priority: iteration % 10 === 0 ? "high" : "normal",
  };
}

function buildIdempotencyKey(iteration) {
  return `lt-${TEST_RUN_ID}-vu${__VU}-it${iteration}-${Date.now()}`;
}

export default function () {
  const payload = buildPayload(__ITER);
  const idempotencyKey = buildIdempotencyKey(__ITER);

  const response = http.post(`${BASE_URL}/send-notification`, JSON.stringify(payload), {
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": idempotencyKey,
    },
    timeout: "20s",
  });

  requestDuration.add(response.timings.duration);

  const ok = check(response, {
    "status is 202 or 200": (r) => r.status === 202 || r.status === 200,
  });

  if (ok) {
    acceptedCount.add(1);
  } else {
    sendErrors.add(1);
  }
}
