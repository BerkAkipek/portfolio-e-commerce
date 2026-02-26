import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const limit = searchParams.get("limit") ?? "100";
  const offset = searchParams.get("offset") ?? "0";

  const upstreamURL = new URL("/products", getApiBaseURL());
  upstreamURL.searchParams.set("limit", limit);
  upstreamURL.searchParams.set("offset", offset);

  try {
    const response = await fetch(upstreamURL, {
      method: "GET",
      cache: "no-store",
      headers: { Accept: "application/json" },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: "failed to fetch products" },
        { status: response.status },
      );
    }

    const data = (await response.json()) as unknown;
    return NextResponse.json(data, { status: 200 });
  } catch {
    return NextResponse.json(
      { error: "unable to reach product service" },
      { status: 502 },
    );
  }
}
