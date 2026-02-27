import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.API_URL?.trim() ?? process.env.NEXT_PUBLIC_API_URL?.trim();
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

    const payload = await response.text();
    const proxied = new NextResponse(payload, { status: response.status });
    const contentType = response.headers.get("content-type");
    if (contentType) {
      proxied.headers.set("content-type", contentType);
    }
    return proxied;
  } catch {
    return NextResponse.json(
      { error: "unable to reach product service" },
      { status: 502 },
    );
  }
}
