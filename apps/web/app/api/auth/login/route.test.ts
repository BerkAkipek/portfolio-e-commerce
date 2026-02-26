import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { POST } from "./route";

describe("POST /api/auth/login", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("proxies request body and cookies to upstream login endpoint", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ success: true }), {
        status: 200,
        headers: {
          "content-type": "application/json",
          "set-cookie": "access_token=abc; Path=/, refresh_token=def; Path=/",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchSpy);

    const request = new NextRequest("http://localhost/api/auth/login", {
      method: "POST",
      headers: { cookie: "guest_session_id=xyz; Path=/" },
      body: JSON.stringify({ email: "user@example.com", password: "P@ssw0rd!" }),
    });
    const response = await POST(request);

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    expect(String(fetchSpy.mock.calls[0][0])).toBe("http://localhost:8080/auth/login");
    expect(fetchSpy.mock.calls[0][1]?.headers).toMatchObject({
      Cookie: "guest_session_id=xyz; Path=/",
    });
    expect(response.status).toBe(200);
    expect(response.headers.get("set-cookie")).toContain("access_token=abc");
    await expect(response.json()).resolves.toEqual({ success: true });
  });
});
