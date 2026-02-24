"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  fetchAllUsage,
  fetchAllLimits,
  setLimits,
  suspendUser,
} from "@/lib/api";
import { isLoggedIn, isAdmin, getUserId, clearSession } from "@/lib/auth";

interface ModelUsage {
  prompt_tokens: number;
  completion_tokens: number;
}

interface UserRow {
  userId: string;
  usage: Record<string, ModelUsage>;
}

interface Toast {
  msg: string;
  ok: boolean;
}

interface LimitForm {
  rps: string;
  max_tokens: string;
  max_tokens_per_request: string;
}

const DEFAULT_FORM: LimitForm = {
  rps: "",
  max_tokens: "",
  max_tokens_per_request: "",
};

export default function AdminDashboard() {
  const router = useRouter();
  const [rows, setRows] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState<Toast | null>(null);
  const [forms, setForms] = useState<Record<string, LimitForm>>({});
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const [currentUser, setCurrentUser] = useState<string | null>(null);
  const [currentLimits, setCurrentLimits] = useState<
    Record<
      string,
      { rps: number; max_tokens: number; max_tokens_per_request: number }
    >
  >({});

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    if (!isAdmin()) {
      router.replace("/chat");
      return;
    }
    setCurrentUser(getUserId());
  }, [router]);

  const handleLogout = () => {
    clearSession();
    router.replace("/login");
  };

  const showToast = (msg: string, ok: boolean) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 3500);
  };

  const load = useCallback(async () => {
    try {
      const [allUsage, allLimits] = await Promise.all([
        fetchAllUsage(),
        fetchAllLimits(),
      ]);
      const userRows: UserRow[] = Object.entries(allUsage).map(
        ([userId, usage]) => ({ userId, usage }),
      );
      setRows(userRows.length > 0 ? userRows : []);
      // Normalise limiter field names to match our form shape
      setCurrentLimits(
        Object.fromEntries(
          Object.entries(allLimits).map(([uid, l]) => [
            uid,
            {
              rps: l.RPS,
              max_tokens: l.MaxTokens,
              max_tokens_per_request: l.MaxTokensPerReq,
            },
          ]),
        ),
      );
    } catch (e) {
      showToast(String(e), false);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleSetLimits = async (userId: string) => {
    const f = forms[userId] ?? DEFAULT_FORM;
    setPending((p) => ({ ...p, [userId]: true }));
    try {
      await setLimits({
        user_id: userId,
        rps: Number(f.rps),
        max_tokens: Number(f.max_tokens),
        max_tokens_per_request: Number(f.max_tokens_per_request),
      });
      showToast(`Limits updated for ${userId}`, true);
      setForms((prev) => ({ ...prev, [userId]: DEFAULT_FORM }));
      // Refresh limits to reflect new values
      fetchAllLimits()
        .then((allLimits) =>
          setCurrentLimits(
            Object.fromEntries(
              Object.entries(allLimits).map(([uid, l]) => [
                uid,
                {
                  rps: l.RPS,
                  max_tokens: l.MaxTokens,
                  max_tokens_per_request: l.MaxTokensPerReq,
                },
              ]),
            ),
          ),
        )
        .catch(() => {});
    } catch (e) {
      showToast(String(e), false);
    } finally {
      setPending((p) => ({ ...p, [userId]: false }));
    }
  };

  const handleSuspend = async (userId: string) => {
    if (!confirm(`Suspend ${userId}?`)) return;
    setPending((p) => ({ ...p, [`${userId}:suspend`]: true }));
    try {
      await suspendUser({ user_id: userId });
      showToast(`${userId} suspended`, true);
    } catch (e) {
      showToast(String(e), false);
    } finally {
      setPending((p) => ({ ...p, [`${userId}:suspend`]: false }));
    }
  };

  const setField = (userId: string, field: keyof LimitForm, val: string) =>
    setForms((prev) => ({
      ...prev,
      [userId]: { ...(prev[userId] ?? DEFAULT_FORM), [field]: val },
    }));

  return (
    <main className="min-h-screen p-8" style={{ background: "var(--bg)" }}>
      {/* Header */}
      <div className="mb-8 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: "var(--purple)" }}>
            🔮 Admin Dashboard
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--muted)" }}>
            Ollama OpenAI-Compatible Proxy — admin panel
          </p>
        </div>
        <div className="flex items-center gap-3">
          {currentUser && (
            <span
              className="text-xs px-3 py-1 rounded-full"
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
      </div>

      {/* Toast */}
      {toast && (
        <div
          className="fixed bottom-6 right-6 px-4 py-2 rounded-lg text-sm font-medium shadow-lg z-50 transition-all"
          style={{
            background: toast.ok ? "var(--green-bg)" : "var(--red-bg)",
            color: toast.ok ? "var(--green-text)" : "var(--red-text)",
            border: `1px solid ${toast.ok ? "var(--green-border)" : "var(--red-border)"}`,
          }}
        >
          {toast.msg}
        </div>
      )}

      {loading ? (
        <p style={{ color: "var(--muted)" }}>Loading…</p>
      ) : (
        rows.map((row) => (
          <UserCard
            key={row.userId}
            row={row}
            form={forms[row.userId] ?? DEFAULT_FORM}
            currentLimit={currentLimits[row.userId]}
            isPending={!!pending[row.userId]}
            isSuspending={!!pending[`${row.userId}:suspend`]}
            onFieldChange={(f, v) => setField(row.userId, f, v)}
            onSetLimits={() => handleSetLimits(row.userId)}
            onSuspend={() => handleSuspend(row.userId)}
          />
        ))
      )}

      <button
        onClick={load}
        className="mt-6 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
        style={{
          background: "var(--surface)",
          color: "var(--purple-light)",
          border: "1px solid var(--border)",
        }}
      >
        ↻ Refresh
      </button>
    </main>
  );
}

function UserCard({
  row,
  form,
  currentLimit,
  isPending,
  isSuspending,
  onFieldChange,
  onSetLimits,
  onSuspend,
}: {
  row: UserRow;
  form: LimitForm;
  currentLimit?: {
    rps: number;
    max_tokens: number;
    max_tokens_per_request: number;
  };
  isPending: boolean;
  isSuspending: boolean;
  onFieldChange: (f: keyof LimitForm, v: string) => void;
  onSetLimits: () => void;
  onSuspend: () => void;
}) {
  const totalTokens = Object.values(row.usage).reduce(
    (sum, u) => sum + u.prompt_tokens + u.completion_tokens,
    0,
  );

  return (
    <div
      className="rounded-xl p-6 mb-6"
      style={{
        background: "var(--surface)",
        border: "1px solid var(--border)",
      }}
    >
      {/* User header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <span
            className="px-3 py-1 rounded-full text-xs font-semibold"
            style={{ background: "#3b0764", color: "#c4b5fd" }}
          >
            {row.userId}
          </span>
          <span className="text-sm" style={{ color: "var(--muted)" }}>
            {totalTokens.toLocaleString()} tokens total
          </span>
        </div>
        <button
          onClick={onSuspend}
          disabled={isSuspending}
          className="px-3 py-1 rounded-lg text-xs font-medium transition-colors disabled:opacity-50"
          style={{
            background: "var(--red-bg)",
            color: "var(--red-text)",
            border: "1px solid var(--red-border)",
          }}
        >
          {isSuspending ? "Suspending…" : "Suspend"}
        </button>
      </div>

      {/* Usage table */}
      <table className="w-full text-sm mb-6">
        <thead>
          <tr
            style={{
              color: "var(--muted)",
              borderBottom: "1px solid var(--border)",
            }}
          >
            <th className="text-left pb-2 font-medium">Model</th>
            <th className="text-right pb-2 font-medium">Prompt</th>
            <th className="text-right pb-2 font-medium">Completion</th>
            <th className="text-right pb-2 font-medium">Total</th>
          </tr>
        </thead>
        <tbody>
          {Object.entries(row.usage).length === 0 ? (
            <tr>
              <td
                colSpan={4}
                className="py-3 text-center"
                style={{ color: "var(--muted)" }}
              >
                No usage recorded
              </td>
            </tr>
          ) : (
            Object.entries(row.usage).map(([model, u]) => (
              <tr
                key={model}
                style={{ borderBottom: "1px solid var(--border)" }}
              >
                <td
                  className="py-2 font-mono"
                  style={{ color: "var(--purple-light)" }}
                >
                  {model}
                </td>
                <td className="py-2 text-right">
                  {u.prompt_tokens.toLocaleString()}
                </td>
                <td className="py-2 text-right">
                  {u.completion_tokens.toLocaleString()}
                </td>
                <td className="py-2 text-right font-medium">
                  {(u.prompt_tokens + u.completion_tokens).toLocaleString()}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>

      {/* Set Limits form */}
      <div>
        <p
          className="text-xs font-semibold mb-2"
          style={{ color: "var(--purple-light)" }}
        >
          Set Limits
        </p>
        <div className="flex gap-3 flex-wrap">
          {[
            { field: "rps" as const, label: "RPS", current: currentLimit?.rps },
            {
              field: "max_tokens" as const,
              label: "Max Tokens",
              current: currentLimit?.max_tokens,
            },
            {
              field: "max_tokens_per_request" as const,
              label: "Per Request",
              current: currentLimit?.max_tokens_per_request,
            },
          ].map(({ field, label, current }) => (
            <div key={field} className="flex flex-col gap-1">
              <label className="text-xs" style={{ color: "var(--muted)" }}>
                {label}
                {current !== undefined && current !== -1 && (
                  <span
                    className="ml-1 font-mono"
                    style={{ color: "var(--purple-light)" }}
                  >
                    (now: {current})
                  </span>
                )}
                {current === -1 && (
                  <span className="ml-1" style={{ color: "var(--muted)" }}>
                    (∞)
                  </span>
                )}
              </label>
              <input
                type="number"
                min={1}
                value={form[field]}
                onChange={(e) => onFieldChange(field, e.target.value)}
                placeholder={
                  current !== undefined && current !== -1
                    ? String(current)
                    : "∞"
                }
                className="w-28 px-2 py-1 rounded-md text-sm outline-none"
                style={{
                  background: "var(--bg)",
                  border: "1px solid var(--border)",
                  color: "var(--text)",
                }}
              />
            </div>
          ))}
          <div className="flex items-end">
            <button
              onClick={onSetLimits}
              disabled={isPending}
              className="px-4 py-1 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
              style={{ background: "var(--purple)", color: "#fff" }}
            >
              {isPending ? "Saving…" : "Apply"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
