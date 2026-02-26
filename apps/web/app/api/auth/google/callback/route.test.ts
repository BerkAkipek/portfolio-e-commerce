import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GET } from "./route";

describe("GET /api/auth/google/callback", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("passes redirect and auth cookies through callback", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(null, {
          status: 302,
          headers: {
            location: "/catalog",
            "set-cookie": "access_token=1; Path=/, refresh_token=2; Path=/",
          },
        }),
      ),
    );

    const request = new NextRequest(
      "http://localhost/api/auth/google/callback?code=test-code&state=state-1",
    );
    const response = await GET(request);

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toBe("http://localhost/catalog");
    expect(response.headers.get("set-cookie")).toContain("refresh_token=2");
  });

  it("passes upstream non-redirect callback response as-is", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "google_state" }), {
          status: 400,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const request = new NextRequest(
      "http://localhost/api/auth/google/callback?code=test-code&state=state-1",
    );
    const response = await GET(request);

    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toEqual({ error: "google_state" });
  });
});
