"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

type AuthMode = "login" | "register";

type AuthResponse = {
  success?: boolean;
  error?: string;
};

async function submitLogin(email: string, password: string): Promise<AuthResponse> {
  const response = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const payload = (await response.json()) as AuthResponse;
  if (!response.ok) {
    throw new Error(payload.error ?? "login failed");
  }
  return payload;
}

async function submitRegister(
  fullName: string,
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ full_name: fullName, email, password }),
  });
  const payload = (await response.json()) as AuthResponse;
  if (!response.ok) {
    throw new Error(payload.error ?? "registration failed");
  }
  return payload;
}

export default function AuthClient() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const [mode, setMode] = useState<AuthMode>("login");
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");

  const mutation = useMutation({
    mutationFn: async () => {
      if (mode === "login") {
        return submitLogin(email, password);
      }
      return submitRegister(fullName, email, password);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["auth-session"] });
      setMessage(
        mode === "login"
          ? "Login successful. Redirecting to catalog..."
          : "Registration successful. Redirecting to catalog...",
      );
      router.push("/catalog");
      router.refresh();
    },
    onError: (error) => {
      setMessage(error instanceof Error ? error.message : "Authentication failed.");
    },
  });

  return (
    <main className="relative mx-auto flex w-full max-w-6xl items-center justify-center px-6 pb-14 pt-10 sm:px-10 lg:px-12">
      <section className="w-full max-w-xl rounded-3xl border border-white/12 bg-white/5 p-6 backdrop-blur">
        <div className="mb-6 flex items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold tracking-[0.18em] text-cyan-200 uppercase">
              Account
            </p>
            <h1 className="mt-1 text-2xl font-semibold text-white sm:text-3xl">
              Register or Login
            </h1>
          </div>
          <Link
            href="/catalog"
            className="rounded-full border border-white/20 px-4 py-2 text-xs font-semibold text-slate-200 transition hover:border-cyan-300/80 hover:text-white"
          >
            Back to catalog
          </Link>
        </div>

        <div className="mb-5 flex rounded-full border border-white/10 bg-black/20 p-1">
          <button
            type="button"
            onClick={() => setMode("login")}
            className={`flex-1 rounded-full px-4 py-2 text-sm font-semibold transition ${
              mode === "login"
                ? "bg-cyan-300 text-[#03151b]"
                : "text-slate-300 hover:text-white"
            }`}
          >
            Login
          </button>
          <button
            type="button"
            onClick={() => setMode("register")}
            className={`flex-1 rounded-full px-4 py-2 text-sm font-semibold transition ${
              mode === "register"
                ? "bg-cyan-300 text-[#03151b]"
                : "text-slate-300 hover:text-white"
            }`}
          >
            Register
          </button>
        </div>

        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            setMessage("");
            mutation.mutate();
          }}
        >
          {mode === "register" ? (
            <label className="block text-sm">
              <span className="mb-1.5 block text-slate-300">Full name</span>
              <input
                value={fullName}
                onChange={(event) => setFullName(event.target.value)}
                required
                className="h-11 w-full rounded-xl border border-white/15 bg-black/30 px-4 text-white outline-none placeholder:text-slate-400 focus:border-cyan-300/70"
                placeholder="Jane Doe"
              />
            </label>
          ) : null}

          <label className="block text-sm">
            <span className="mb-1.5 block text-slate-300">Email</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
              className="h-11 w-full rounded-xl border border-white/15 bg-black/30 px-4 text-white outline-none placeholder:text-slate-400 focus:border-cyan-300/70"
              placeholder="you@example.com"
            />
          </label>

          <label className="block text-sm">
            <span className="mb-1.5 block text-slate-300">Password</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
              minLength={8}
              className="h-11 w-full rounded-xl border border-white/15 bg-black/30 px-4 text-white outline-none placeholder:text-slate-400 focus:border-cyan-300/70"
              placeholder="At least 8 characters"
            />
          </label>

          <button
            type="submit"
            disabled={mutation.isPending}
            className="mt-2 w-full rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold tracking-wide text-[#03151b] transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {mutation.isPending
              ? "Submitting..."
              : mode === "login"
                ? "Login"
                : "Create account"}
          </button>
        </form>

        <div className="mt-5">
          <Link
            href="/api/auth/google/start"
            className="inline-flex w-full items-center justify-center rounded-full border border-white/20 px-5 py-3 text-sm font-semibold tracking-wide text-slate-100 transition hover:border-cyan-300/80 hover:text-white"
          >
            Continue with Google
          </Link>
        </div>

        {message ? <p className="mt-4 text-sm text-slate-200">{message}</p> : null}
      </section>
    </main>
  );
}
