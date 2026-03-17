import { NextRequest, NextResponse } from "next/server";

const apiBaseUrl = process.env.NOTIFICATION_API_BASE_URL ?? "http://localhost:8080";

export async function GET(request: NextRequest) {
  try {
    const limit = request.nextUrl.searchParams.get("limit") ?? "10";
    const response = await fetch(`${apiBaseUrl}/notifications/recent?limit=${limit}`, {
      cache: "no-store",
      headers: {
        Accept: "application/json",
      },
    });

    const data = await response.json().catch(() => []);
    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json(
      { error: "Unable to reach backend API" },
      { status: 503 },
    );
  }
}
