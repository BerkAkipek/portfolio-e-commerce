import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GET } from "./route";

describe("GET /api/auth/session", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reports authenticated=false on upstream 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "unauthorized" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/session");
    const response = await GET(request);

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      authenticated: false,
      restored: false,
    });
  });

  it("reports restored=true when rotated auth cookies are returned", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ authenticated: true }), {
          status: 200,
          headers: {
            "content-type": "application/json",
            "set-cookie": "access_token=rotated-a; Path=/, refresh_token=rotated-r; Path=/",
          },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/session");
    const response = await GET(request);

    expect(response.status).toBe(200);
    expect(response.headers.get("set-cookie")).toContain("access_token=rotated-a");
    await expect(response.json()).resolves.toEqual({
      authenticated: true,
      restored: true,
    });
  });

  it("reports authenticated=false on non-401 upstream errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "upstream failure" }), {
          status: 500,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/session");
    const response = await GET(request);

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      authenticated: false,
      restored: false,
    });
  });

  it("returns 502 when upstream session probe fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    const request = new NextRequest("http://localhost/api/auth/session");
    const response = await GET(request);

    expect(response.status).toBe(502);
    await expect(response.json()).resolves.toEqual({
      error: "unable to probe auth session",
    });
  });
});
