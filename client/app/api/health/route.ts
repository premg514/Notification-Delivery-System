import { NextResponse } from "next/server";

const apiBaseUrl = process.env.NOTIFICATION_API_BASE_URL ?? "http://localhost:8080";

export async function GET() {
  try {
    const response = await fetch(`${apiBaseUrl}/health`, {
      cache: "no-store",
      headers: {
        Accept: "application/json",
      },
    });

    const data = await response.json().catch(() => ({ status: "down" }));
    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json(
      { status: "down", error: "Backend is unreachable" },
      { status: 503 },
    );
  }
}
