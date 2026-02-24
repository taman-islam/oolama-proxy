"use client";

import { useState, FormEvent } from "react";
import { useRouter } from "next/navigation";
import { login, saveSession } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const { user_id, api_key, is_admin } = await login(username, password);
      saveSession(user_id, api_key, is_admin);
      router.push(is_admin ? "/" : "/chat");
    } catch (e) {
      setError(String(e).replace("Error: ", ""));
    } finally {
      setLoading(false);
    }
  };

  return (
    <main
      className="min-h-screen flex items-center justify-center p-4"
      style={{ background: "var(--bg)" }}
    >
      <div
        className="w-full max-w-sm rounded-2xl p-8"
        style={{
          background: "var(--surface)",
          border: "1px solid var(--border)",
        }}
      >
        {/* Logo / title */}
        <div className="text-center mb-8">
          <div className="text-4xl mb-3">🔮</div>
          <h1 className="text-xl font-bold" style={{ color: "var(--purple)" }}>
            Proxy Admin
          </h1>
          <p className="text-xs mt-1" style={{ color: "var(--muted)" }}>
            Sign in to continue
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label
              className="text-xs font-medium"
              style={{ color: "var(--muted)" }}
            >
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="alice"
              autoComplete="username"
              required
              className="px-3 py-2 rounded-lg text-sm outline-none focus:ring-2"
              style={{
                background: "var(--bg)",
                border: "1px solid var(--border)",
                color: "var(--text)",
                // @ts-expect-error css var
                "--tw-ring-color": "var(--purple)",
              }}
            />
          </div>

          <div className="flex flex-col gap-1">
            <label
              className="text-xs font-medium"
              style={{ color: "var(--muted)" }}
            >
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="current-password"
              required
              className="px-3 py-2 rounded-lg text-sm outline-none focus:ring-2"
              style={{
                background: "var(--bg)",
                border: "1px solid var(--border)",
                color: "var(--text)",
              }}
            />
          </div>

          {error && (
            <p
              className="text-xs px-3 py-2 rounded-lg"
              style={{
                background: "var(--red-bg)",
                color: "var(--red-text)",
                border: "1px solid var(--red-border)",
              }}
            >
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="mt-2 py-2 rounded-lg text-sm font-semibold transition-opacity disabled:opacity-50"
            style={{ background: "var(--purple)", color: "#fff" }}
          >
            {loading ? "Signing in…" : "Sign in"}
          </button>
        </form>

        <p
          className="text-center text-xs mt-6"
          style={{ color: "var(--muted)" }}
        >
          Demo users: alice / bob / charlie
        </p>
      </div>
    </main>
  );
}
