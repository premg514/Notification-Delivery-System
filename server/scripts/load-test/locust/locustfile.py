import logging
import os
import random
import time
import uuid

from locust import HttpUser, between, events, task

MIN_WAIT_MS = max(0, int(os.getenv("MIN_WAIT_MS", "50")))
MAX_WAIT_MS = max(MIN_WAIT_MS, int(os.getenv("MAX_WAIT_MS", "250")))
REQUEST_TIMEOUT_SECONDS = max(1, int(os.getenv("REQUEST_TIMEOUT_SECONDS", "15")))
TEST_RUN_ID = os.getenv("TEST_RUN_ID", "local")
DEPARTMENTS = ["CSE", "ECE", "ME", "CIVIL", "EEE"]


class NotificationUser(HttpUser):
    wait_time = between(MIN_WAIT_MS / 1000.0, MAX_WAIT_MS / 1000.0)

    @task
    def send_notification(self):
        iteration = int(time.time() * 1000)
        runner = self.environment.runner
        active_users = runner.user_count if runner is not None else 0
        idempotency_key = (
            f"lt-{TEST_RUN_ID}-u{active_users}-"
            f"{uuid.uuid4()}"
        )

        payload = {
            "title": f"Load Test Notification {iteration}",
            "message": "Training session starts tomorrow at 9 AM.",
            "target_department": random.choice(DEPARTMENTS),
            "priority": "high" if random.random() < 0.1 else "normal",
        }

        headers = {
            "Content-Type": "application/json",
            "Idempotency-Key": idempotency_key,
        }

        with self.client.post(
            "/send-notification",
            json=payload,
            headers=headers,
            name="/send-notification",
            timeout=REQUEST_TIMEOUT_SECONDS,
            catch_response=True,
        ) as response:
            if response.status_code in (200, 202):
                response.success()
                return

            snippet = response.text[:300] if response.text else ""
            response.failure(
                f"unexpected status={response.status_code} body={snippet}"
            )


@events.test_start.add_listener
def on_test_start(environment, **_kwargs):
    logging.getLogger(__name__).info("Loaded %d departments for Locust run", len(DEPARTMENTS))
