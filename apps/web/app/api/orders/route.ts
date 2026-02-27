import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.API_URL?.trim() ?? process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const limit = searchParams.get("limit") ?? "20";
  const offset = searchParams.get("offset") ?? "0";
  const upstreamURL = new URL("/orders", getApiBaseURL());
  upstreamURL.searchParams.set("limit", limit);
  upstreamURL.searchParams.set("offset", offset);

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
