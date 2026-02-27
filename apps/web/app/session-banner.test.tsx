import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import SessionBanner from "./session-banner";

function renderBanner() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <SessionBanner />
    </QueryClientProvider>,
  );
}

describe("SessionBanner", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows restored session messaging when refresh token rotation occurs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ authenticated: true, restored: true }),
      }),
    );

    renderBanner();

    expect(await screen.findByText("Signed in")).toBeInTheDocument();
    expect(
      screen.getByText("Session restored securely via refresh token rotation."),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Security: sessions expire quickly, refresh silently, and logout revokes refresh tokens.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Logout" })).toBeInTheDocument();
  });

  it("shows auth call-to-action for guest users", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ authenticated: false, restored: false }),
      }),
    );

    renderBanner();

    expect(await screen.findByText("Guest session")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Register / Login" })).toBeInTheDocument();
  });

  it("logs out and updates status", async () => {
    const fetchSpy = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ authenticated: true, restored: false }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ authenticated: false, restored: false }),
      });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderBanner();

    await screen.findByText("Signed in");
    await user.click(screen.getByRole("button", { name: "Logout" }));
    expect(await screen.findByText("Signed out.")).toBeInTheDocument();
    expect(await screen.findByText("Guest session")).toBeInTheDocument();
  });

  it("shows retry guidance when auth session refresh cannot reach network", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );

    renderBanner();

    expect(await screen.findByText("Session unavailable")).toBeInTheDocument();
    expect(
      screen.getByText("Offline. Cannot refresh authentication state."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
