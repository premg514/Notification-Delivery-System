import json
import logging
import os
import random
import time
import uuid
from pathlib import Path

from locust import HttpUser, between, events, task

ROOT = Path(__file__).resolve().parent.parent
USERS_FILE = ROOT / "data" / "users.json"

TARGET_USERS_PER_REQUEST = max(1, int(os.getenv("TARGET_USERS_PER_REQUEST", "50")))
MIN_WAIT_MS = max(0, int(os.getenv("MIN_WAIT_MS", "50")))
MAX_WAIT_MS = max(MIN_WAIT_MS, int(os.getenv("MAX_WAIT_MS", "250")))
REQUEST_TIMEOUT_SECONDS = max(1, int(os.getenv("REQUEST_TIMEOUT_SECONDS", "15")))
TEST_RUN_ID = os.getenv("TEST_RUN_ID", "local")


def _load_users():
    if not USERS_FILE.exists():
        raise FileNotFoundError(
            f"Missing user data file: {USERS_FILE}. "
            "Create it as a JSON array of user UUID strings."
        )

    data = json.loads(USERS_FILE.read_text(encoding="utf-8"))
    if not isinstance(data, list) or len(data) == 0:
        raise ValueError(f"{USERS_FILE} must contain a non-empty JSON array")

    return data


USER_IDS = _load_users()


def pick_target_users():
    if TARGET_USERS_PER_REQUEST >= len(USER_IDS):
        return USER_IDS.copy()

    return random.sample(USER_IDS, TARGET_USERS_PER_REQUEST)


class NotificationUser(HttpUser):
    wait_time = between(MIN_WAIT_MS / 1000.0, MAX_WAIT_MS / 1000.0)

    @task
    def send_notification(self):
        target_users = pick_target_users()
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
            "target_users": target_users,
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
    logging.getLogger(__name__).info("Loaded %d user IDs for Locust run", len(USER_IDS))
