import { NextRequest, NextResponse } from "next/server";

import { getApiBaseURL } from "../../_utils";

export async function GET(request: NextRequest) {
  const upstreamURL = new URL("/auth/google/callback", getApiBaseURL());
  const code = request.nextUrl.searchParams.get("code");
  const state = request.nextUrl.searchParams.get("state");
  if (code) {
    upstreamURL.searchParams.set("code", code);
  }
  if (state) {
    upstreamURL.searchParams.set("state", state);
  }

  try {
    const response = await fetch(upstreamURL, {
      method: "GET",
      redirect: "manual",
      cache: "no-store",
      headers: {
        Cookie: request.headers.get("cookie") ?? "",
      },
    });

    const location = response.headers.get("location");
    if (!location) {
      const payload = await response.text();
      const proxied = new NextResponse(payload, { status: response.status });
      const contentType = response.headers.get("content-type");
      if (contentType) {
        proxied.headers.set("content-type", contentType);
      }
      return proxied;
    }

    const redirectURL = new URL(location, request.nextUrl.origin);
    const proxied = NextResponse.redirect(redirectURL, response.status);
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
