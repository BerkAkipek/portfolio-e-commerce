"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";

type SessionState = {
  authenticated: boolean;
  restored: boolean;
};

async function fetchSessionState(): Promise<SessionState> {
  const response = await fetch("/api/auth/session", {
    method: "GET",
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error("failed to load session");
  }
  return (await response.json()) as SessionState;
}

async function logout() {
  const response = await fetch("/api/auth/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
  if (!response.ok) {
    throw new Error("failed to logout");
  }
}

export default function SessionBanner() {
  const queryClient = useQueryClient();
  const [message, setMessage] = useState("");

  const sessionQuery = useQuery({
    queryKey: ["auth-session"],
    queryFn: fetchSessionState,
    retry: false,
  });

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      setMessage("Signed out.");
      await queryClient.invalidateQueries({ queryKey: ["auth-session"] });
    },
    onError: (error) => {
      setMessage(error instanceof Error ? error.message : "Unable to sign out.");
    },
  });

  const isAuthenticated = sessionQuery.data?.authenticated ?? false;
  const restoredMessage = sessionQuery.data?.restored
    ? "Session restored securely via refresh token rotation."
    : "";

  return (
    <div className="relative z-30 border-b border-white/10 bg-black/25 backdrop-blur">
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-3 px-6 py-3 text-xs sm:px-10 lg:px-12">
        <div>
          <div className="flex items-center gap-2">
            <span
              className={`inline-flex rounded-full px-2.5 py-1 font-semibold tracking-wide uppercase ${
                isAuthenticated
                  ? "bg-emerald-300/20 text-emerald-100"
                  : "bg-slate-500/20 text-slate-200"
              }`}
            >
              {isAuthenticated ? "Signed in" : "Guest session"}
            </span>
            {restoredMessage ? (
              <span className="text-cyan-100">{restoredMessage}</span>
            ) : message ? (
              <span className="text-slate-200">{message}</span>
            ) : null}
          </div>
          <p className="mt-1 text-[11px] text-slate-300">
            Security: sessions expire quickly, refresh silently, and logout revokes refresh tokens.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {!isAuthenticated ? (
            <Link
              href="/auth"
              className="rounded-full border border-white/25 px-3 py-1.5 font-semibold text-slate-100 transition hover:border-cyan-300/80 hover:text-white"
            >
              Register / Login
            </Link>
          ) : (
            <>
              <Link
                href="/orders"
                className="rounded-full border border-white/25 px-3 py-1.5 font-semibold text-slate-100 transition hover:border-cyan-300/80 hover:text-white"
              >
                My Orders
              </Link>
              <button
                type="button"
                onClick={() => logoutMutation.mutate()}
                disabled={logoutMutation.isPending}
                className="rounded-full border border-white/25 px-3 py-1.5 font-semibold text-slate-100 transition hover:border-red-300/80 hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
              >
                {logoutMutation.isPending ? "Signing out..." : "Logout"}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
