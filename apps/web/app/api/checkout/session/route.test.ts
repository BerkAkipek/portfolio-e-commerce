import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { POST } from "./route";

describe("POST /api/checkout/session", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("proxies checkout session response with cookies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: { session_id: "cs_1", url: "https://checkout.stripe.com/c/pay/cs_1" } }), {
          status: 200,
          headers: {
            "content-type": "application/json",
            "set-cookie": "access_token=new; Path=/, refresh_token=new2; Path=/",
          },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/checkout/session", {
      method: "POST",
      body: JSON.stringify({
        success_url: "http://localhost:3000/checkout/success",
        cancel_url: "http://localhost:3000/checkout/cancel",
      }),
    });
    const response = await POST(request);

    expect(response.status).toBe(200);
    expect(response.headers.get("set-cookie")).toContain("refresh_token=new2");
    await expect(response.json()).resolves.toEqual({
      data: {
        session_id: "cs_1",
        url: "https://checkout.stripe.com/c/pay/cs_1",
      },
    });
  });
});
