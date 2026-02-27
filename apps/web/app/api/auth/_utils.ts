import { NextRequest, NextResponse } from "next/server";

export function getApiBaseURL(): string {
  const base = process.env.API_URL?.trim() ?? process.env.NEXT_PUBLIC_API_URL?.trim();
  return base && base.length > 0 ? base : "http://localhost:8080";
}

function getCSRFHeader(request: NextRequest): Record<string, string> {
  const csrfToken = request.cookies.get("csrf_token")?.value?.trim();
  if (!csrfToken) {
    return {};
  }
  return { "X-CSRF-Token": csrfToken };
}

export async function proxyJSONPost(request: NextRequest, path: string) {
  const upstreamURL = new URL(path, getApiBaseURL());

  try {
    const rawBody = await request.text();
    const response = await fetch(upstreamURL, {
      method: "POST",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
        ...getCSRFHeader(request),
      },
      body: rawBody,
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
      { error: "unable to reach auth service" },
      { status: 502 },
    );
  }
}
