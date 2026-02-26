import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { POST } from "./route";

describe("POST /api/auth/register", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("proxies registration response and cookies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ success: true }), {
          status: 200,
          headers: {
            "content-type": "application/json",
            "set-cookie": "access_token=abc; Path=/, refresh_token=def; Path=/",
          },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/auth/register", {
      method: "POST",
      body: JSON.stringify({
        full_name: "New User",
        email: "new@example.com",
        password: "P@ssw0rd!",
      }),
    });
    const response = await POST(request);

    expect(response.status).toBe(200);
    expect(response.headers.get("set-cookie")).toContain("refresh_token=def");
    await expect(response.json()).resolves.toEqual({ success: true });
  });
});
