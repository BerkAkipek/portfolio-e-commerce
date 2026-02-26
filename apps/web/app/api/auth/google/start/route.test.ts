import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GET } from "./route";

describe("GET /api/auth/google/start", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("passes upstream redirect and oauth cookies through", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(null, {
          status: 302,
          headers: {
            location: "https://accounts.google.com/o/oauth2/v2/auth?state=abc",
            "set-cookie": "google_oauth_state=abc; Path=/; HttpOnly",
          },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/google/start");
    const response = await GET(request);

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("accounts.google.com");
    expect(response.headers.get("set-cookie")).toContain("google_oauth_state=abc");
  });

  it("passes through non-redirect upstream errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "google oauth is not configured" }), {
          status: 503,
          headers: {
            "content-type": "application/json",
          },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/google/start");
    const response = await GET(request);

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toEqual({
      error: "google oauth is not configured",
    });
  });
});
