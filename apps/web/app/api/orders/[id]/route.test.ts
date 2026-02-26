import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GET } from "./route";

describe("GET /api/orders/[id]", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("proxies order detail payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: {
              order: {
                id: "o1",
                user_id: "u1",
                status: "paid",
                currency: "USD",
                subtotal_cents: 1000,
                total_cents: 1200,
                created_at: "2026-02-01T00:00:00Z",
                updated_at: "2026-02-01T00:00:00Z",
              },
              items: [],
            },
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      ),
    );

    const request = new NextRequest("http://localhost/api/orders/o1");
    const response = await GET(request, { params: Promise.resolve({ id: "o1" }) });

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      data: {
        order: {
          id: "o1",
          user_id: "u1",
          status: "paid",
          currency: "USD",
          subtotal_cents: 1000,
          total_cents: 1200,
          created_at: "2026-02-01T00:00:00Z",
          updated_at: "2026-02-01T00:00:00Z",
        },
        items: [],
      },
    });
  });
});
