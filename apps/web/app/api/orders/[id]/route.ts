import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> },
) {
  const { id } = await context.params;
  const upstreamURL = new URL(`/orders/${encodeURIComponent(id)}`, getApiBaseURL());

  try {
    const response = await fetch(upstreamURL, {
      method: "GET",
      cache: "no-store",
      headers: {
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
      },
    });

    const payload = await response.text();
    const proxied = new NextResponse(payload, { status: response.status });
    const contentType = response.headers.get("content-type");
    if (contentType) {
      proxied.headers.set("content-type", contentType);
    }
    const setCookie = response.headers.get("set-cookie");
    if (setCookie) {
      proxied.headers.set("set-cookie", setCookie);
    }
    return proxied;
  } catch {
    return NextResponse.json(
      { error: "unable to reach orders service" },
      { status: 502 },
    );
  }
}
