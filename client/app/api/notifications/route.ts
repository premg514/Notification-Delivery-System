import { NextRequest, NextResponse } from "next/server";

const apiBaseUrl = process.env.NOTIFICATION_API_BASE_URL ?? "http://localhost:8080";

export async function POST(request: NextRequest) {
  try {
    const body = await request.text();
    const idempotencyKey = request.headers.get("Idempotency-Key")?.trim();

    const response = await fetch(`${apiBaseUrl}/send-notification`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}),
      },
      body,
      cache: "no-store",
    });

    const data = await response.json().catch(() => ({
      error: "Unexpected backend response",
    }));

    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json(
      { error: "Unable to reach backend API" },
      { status: 503 },
    );
  }
}
