import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function GET(
  _request: NextRequest,
  context: { params: Promise<{ slug: string }> },
) {
  const { slug } = await context.params;
  const upstreamURL = new URL(`/products/${encodeURIComponent(slug)}`, getApiBaseURL());

  try {
    const response = await fetch(upstreamURL, {
      method: "GET",
      cache: "no-store",
      headers: { Accept: "application/json" },
    });

    const data = (await response.json()) as unknown;
    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json(
      { error: "unable to reach product service" },
      { status: 502 },
    );
  }
}
