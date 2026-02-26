import { afterEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";

import { GET } from "./route";

describe("GET /api/products/[slug]", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("passes through non-JSON upstream responses with original status and content-type", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("upstream failed", {
          status: 503,
          headers: { "content-type": "text/plain; charset=utf-8" },
        }),
      ),
    );

    const request = new NextRequest("http://localhost/api/products/test-slug");
    const response = await GET(request, { params: Promise.resolve({ slug: "test-slug" }) });

    expect(response.status).toBe(503);
    expect(response.headers.get("content-type")).toContain("text/plain");
    expect(await response.text()).toBe("upstream failed");
  });

  it("returns 502 when upstream fetch throws", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    const request = new NextRequest("http://localhost/api/products/test-slug");
    const response = await GET(request, { params: Promise.resolve({ slug: "test-slug" }) });

    expect(response.status).toBe(502);
    await expect(response.json()).resolves.toEqual({
      error: "unable to reach product service",
    });
  });
});
