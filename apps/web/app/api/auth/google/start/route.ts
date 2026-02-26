import { NextRequest, NextResponse } from "next/server";

import { getApiBaseURL } from "../../_utils";

export async function GET(request: NextRequest) {
  const upstreamURL = new URL("/auth/google/start", getApiBaseURL());

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
      return new NextResponse(payload, { status: response.status });
    }

    const proxied = NextResponse.redirect(location, response.status);
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
