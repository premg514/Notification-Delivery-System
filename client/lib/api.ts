export type NotificationPayload = {
  title: string;
  message: string;
  target_department: string;
  priority: string;
};

export type NotificationResponse = {
  notification_id: string;
  status: string;
  queued_deliveries: number;
  target_department: string;
  duplicate: boolean;
  created_at: string;
};

export type RecentNotification = {
  id: string;
  title: string;
  target_department: string;
  priority: string;
  status: string;
  created_at: string;
  queued_deliveries: number;
};

export async function sendNotification(
  payload: NotificationPayload,
  idempotencyKey: string,
) {
  const response = await fetch("/api/notifications", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": idempotencyKey,
    },
    body: JSON.stringify(payload),
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message =
      typeof data?.error === "string" ? data.error : "Failed to send notification";
    throw new Error(message);
  }

  return data as NotificationResponse;
}

export async function getHealth() {
  const response = await fetch("/api/health", { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Health check failed");
  }

  return (await response.json()) as { status: string };
}

export async function getRecentNotifications(
  limit = 10,
): Promise<RecentNotification[]> {
  const response = await fetch(`/api/notifications/recent?limit=${limit}`, {
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error("Failed to fetch recent notifications");
  }

  return (await response.json()) as RecentNotification[];
}
