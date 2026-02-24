"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { fetchMyUsage } from "@/lib/api";
import { isLoggedIn, getUserId, clearSession } from "@/lib/auth";
import { ModelUsage } from "../../generated/api";

export default function UsagePage() {
  const router = useRouter();
  const [usage, setUsage] = useState<{ [key: string]: ModelUsage }>({});
  const [loading, setLoading] = useState(true);
  const [currentUser, setCurrentUser] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    setCurrentUser(getUserId());
  }, [router]);

  const load = useCallback(async () => {
    try {
      const resp = await fetchMyUsage();
      setUsage(resp.usageByModel);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleLogout = () => {
    clearSession();
    router.replace("/login");
  };

  const totalTokens = Object.values(usage).reduce(
    (sum, u) => sum + u.promptTokens + u.completionTokens,
    0,
  );

  return (
    <div className="flex flex-col h-screen" style={{ background: "var(--bg)" }}>
      {/* Header */}
      <header
        className="flex items-center justify-between px-6 py-3 border-b shrink-0"
        style={{ borderColor: "var(--border)", background: "var(--surface)" }}
      >
        <div className="flex items-center gap-4">
          <span
            className="font-semibold text-sm"
            style={{ color: "var(--purple)" }}
          >
            🔮 Proxy Chat
          </span>
          <button
            onClick={() => router.push("/chat")}
            className="text-xs px-3 py-1 rounded-lg transition-colors"
            style={{
              color: "var(--text)",
              background: "var(--bg)",
              border: "1px solid var(--border)",
            }}
          >
            ← Back to Chat
          </button>
        </div>
        <div className="flex items-center gap-3">
          {currentUser && (
            <span
              className="text-xs px-3 py-1 rounded-full font-semibold"
              style={{ background: "#3b0764", color: "#c4b5fd" }}
            >
              {currentUser}
            </span>
          )}
          <button
            onClick={handleLogout}
            className="text-xs px-3 py-1 rounded-lg transition-colors"
            style={{
              background: "var(--surface)",
              color: "var(--muted)",
              border: "1px solid var(--border)",
            }}
          >
            Logout
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto p-8 flex flex-col items-center">
        <div className="w-full max-w-2xl">
          <h1 className="text-xl font-bold mb-6 text-white text-center">
            My Token Usage
          </h1>

          <div
            className="rounded-xl p-6 mb-6"
            style={{
              background: "var(--surface)",
              border: "1px solid var(--border)",
            }}
          >
            <div className="flex items-center justify-between mb-6">
              <span
                className="text-sm font-medium"
                style={{ color: "var(--text)" }}
              >
                Total Consumption
              </span>
              <span
                className="text-sm font-semibold"
                style={{ color: "var(--purple-light)" }}
              >
                {totalTokens.toLocaleString()} tokens
              </span>
            </div>

            {loading ? (
              <p
                className="text-sm text-center py-4"
                style={{ color: "var(--muted)" }}
              >
                Loading...
              </p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr
                    style={{
                      color: "var(--muted)",
                      borderBottom: "1px solid var(--border)",
                    }}
                  >
                    <th className="text-left pb-3 font-medium">Model</th>
                    <th className="text-right pb-3 font-medium">Prompt</th>
                    <th className="text-right pb-3 font-medium">Completion</th>
                    <th className="text-right pb-3 font-medium">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(usage).length === 0 ? (
                    <tr>
                      <td
                        colSpan={4}
                        className="py-6 text-center text-sm"
                        style={{ color: "var(--muted)" }}
                      >
                        No usage recorded yet. Start chatting!
                      </td>
                    </tr>
                  ) : (
                    Object.entries(usage).map(([model, u]) => (
                      <tr
                        key={model}
                        style={{ borderBottom: "1px solid var(--border)" }}
                      >
                        <td
                          className="py-3 font-mono text-xs font-semibold"
                          style={{ color: "var(--purple-light)" }}
                        >
                          {model}
                        </td>
                        <td className="py-3 text-right text-gray-300">
                          {u.promptTokens.toLocaleString()}
                        </td>
                        <td className="py-3 text-right text-gray-300">
                          {u.completionTokens.toLocaleString()}
                        </td>
                        <td className="py-3 text-right font-medium text-white">
                          {(
                            u.promptTokens + u.completionTokens
                          ).toLocaleString()}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}

            <div className="mt-6 flex justify-end">
              <button
                onClick={load}
                className="text-xs px-4 py-2 rounded-lg transition-colors font-medium"
                style={{
                  background: "var(--bg)",
                  color: "var(--purple-light)",
                  border: "1px solid var(--border)",
                }}
              >
                ↻ Refresh
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
