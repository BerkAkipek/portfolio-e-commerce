import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function POST(request: NextRequest) {
  const upstreamURL = new URL("/cart/items", getApiBaseURL());

  try {
    const rawBody = await request.text();
    const response = await fetch(upstreamURL, {
      method: "POST",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
      },
      body: rawBody,
    });

    const payload = await response.text();
    const proxied = new NextResponse(payload, {
      status: response.status,
    });

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
      { error: "unable to reach cart service" },
      { status: 502 },
    );
  }
}
