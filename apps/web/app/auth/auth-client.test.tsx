import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import AuthClient from "./auth-client";

const pushSpy = vi.fn();
const refreshSpy = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushSpy,
    refresh: refreshSpy,
  }),
}));

function renderAuthClient() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthClient />
    </QueryClientProvider>,
  );
}

describe("AuthClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    pushSpy.mockReset();
    refreshSpy.mockReset();
  });

  it("submits login and redirects on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ success: true }),
      }),
    );

    const user = userEvent.setup();
    renderAuthClient();

    await user.type(screen.getByPlaceholderText("you@example.com"), "user@example.com");
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "P@ssw0rd!");
    const loginButtons = screen.getAllByRole("button", { name: "Login" });
    await user.click(loginButtons[1]);

    expect(await screen.findByText("Login successful. Redirecting to catalog...")).toBeInTheDocument();
    expect(pushSpy).toHaveBeenCalledWith("/catalog");
    expect(refreshSpy).toHaveBeenCalledTimes(1);
  });

  it("shows register API error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ error: "email already registered" }),
      }),
    );

    const user = userEvent.setup();
    renderAuthClient();

    await user.click(screen.getByRole("button", { name: "Register" }));
    await user.type(screen.getByPlaceholderText("Jane Doe"), "Existing User");
    await user.type(screen.getByPlaceholderText("you@example.com"), "existing@example.com");
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "P@ssw0rd!");
    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("email already registered")).toBeInTheDocument();
  });

  it("renders google sign-in option", () => {
    renderAuthClient();
    const googleLink = screen.getByRole("link", { name: "Continue with Google" });
    expect(googleLink).toHaveAttribute("href", "/api/auth/google/start");
  });

  it("submits register payload and redirects on success", async () => {
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ success: true }),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    renderAuthClient();

    await user.click(screen.getByRole("button", { name: "Register" }));
    await user.type(screen.getByPlaceholderText("Jane Doe"), "New User");
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "P@ssw0rd!");
    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("Registration successful. Redirecting to catalog...")).toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/auth/register",
      expect.objectContaining({
        method: "POST",
      }),
    );
    const body = (fetchSpy.mock.calls[0]?.[1] as RequestInit).body;
    expect(body).toBe(
      JSON.stringify({
        full_name: "New User",
        email: "new@example.com",
        password: "P@ssw0rd!",
      }),
    );
    expect(pushSpy).toHaveBeenCalledWith("/catalog");
    expect(refreshSpy).toHaveBeenCalled();
  });
});
