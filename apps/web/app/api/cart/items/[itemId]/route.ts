import { NextRequest, NextResponse } from "next/server";

function getApiBaseURL(): string {
  const base = process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

export async function PATCH(
  request: NextRequest,
  context: { params: Promise<{ itemId: string }> },
) {
  const { itemId } = await context.params;
  const upstreamURL = new URL(`/cart/items/${encodeURIComponent(itemId)}`, getApiBaseURL());

  try {
    const rawBody = await request.text();
    const response = await fetch(upstreamURL, {
      method: "PATCH",
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

export async function DELETE(
  request: NextRequest,
  context: { params: Promise<{ itemId: string }> },
) {
  const { itemId } = await context.params;
  const upstreamURL = new URL(`/cart/items/${encodeURIComponent(itemId)}`, getApiBaseURL());

  try {
    const response = await fetch(upstreamURL, {
      method: "DELETE",
      cache: "no-store",
      headers: {
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
      },
    });

    return new NextResponse(null, { status: response.status });
  } catch {
    return NextResponse.json(
      { error: "unable to reach cart service" },
      { status: 502 },
    );
  }
}
