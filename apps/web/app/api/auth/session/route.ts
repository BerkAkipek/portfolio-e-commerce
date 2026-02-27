import { NextRequest, NextResponse } from "next/server";

import { getApiBaseURL } from "../_utils";

type SessionProbeResponse = {
  authenticated: boolean;
  restored: boolean;
};

function hasRotatedAuthCookies(setCookieHeader: string | null): boolean {
  if (!setCookieHeader) {
    return false;
  }
  return (
    setCookieHeader.includes("access_token=") &&
    setCookieHeader.includes("refresh_token=")
  );
}

export async function GET(request: NextRequest) {
  const upstreamURL = new URL("/auth/session", getApiBaseURL());

  try {
    const response = await fetch(upstreamURL, {
      method: "GET",
      cache: "no-store",
      headers: {
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
      },
    });

    const setCookie = response.headers.get("set-cookie");
    const restored = hasRotatedAuthCookies(setCookie);
    const authenticated = response.status === 200;

    const payload: SessionProbeResponse = { authenticated, restored };
    const proxied = NextResponse.json(payload, { status: 200 });
    if (setCookie) {
      proxied.headers.set("set-cookie", setCookie);
    }
    return proxied;
  } catch {
    return NextResponse.json(
      { error: "unable to probe auth session" },
      { status: 502 },
    );
  }
}
