import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GET } from "./route";

describe("GET /api/orders", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("proxies order list response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "o1",
                user_id: "u1",
                status: "paid",
                currency: "USD",
                subtotal_cents: 1000,
                total_cents: 1200,
                created_at: "2026-02-01T00:00:00Z",
                updated_at: "2026-02-01T00:00:00Z",
              },
            ],
            limit: 20,
            offset: 0,
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      ),
    );

    const request = new NextRequest("http://localhost/api/orders?limit=20&offset=0");
    const response = await GET(request);

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      data: [
        {
          id: "o1",
          user_id: "u1",
          status: "paid",
          currency: "USD",
          subtotal_cents: 1000,
          total_cents: 1200,
          created_at: "2026-02-01T00:00:00Z",
          updated_at: "2026-02-01T00:00:00Z",
        },
      ],
      limit: 20,
      offset: 0,
    });
  });

  it("returns 502 when upstream is unreachable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("down")));
    const request = new NextRequest("http://localhost/api/orders");
    const response = await GET(request);
    expect(response.status).toBe(502);
    await expect(response.json()).resolves.toEqual({
      error: "unable to reach orders service",
    });
  });
});
