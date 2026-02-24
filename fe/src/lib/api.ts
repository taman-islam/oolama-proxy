import { getApiKey } from "./auth";
import type { SetLimitsPayload, SuspendPayload } from "./types";

// Admin calls always use the admin key for the /admin/* routes.
const BASE = "http://localhost:8000";

function userHeaders(): HeadersInit {
  const key = getApiKey() ?? "";
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${key}`,
  };
}

/** Fetch usage for the authenticated user. */
export async function fetchMyUsage() {
  const res = await fetch(`${BASE}/v1/usage`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`My usage fetch failed: ${res.status}`);
  return res.json() as Promise<
    Record<string, { prompt_tokens: number; completion_tokens: number }>
  >;
}

/** Fetch usage for ALL users via the admin endpoint. */
export async function fetchAllUsage() {
  const res = await fetch(`${BASE}/admin/usage`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`Usage fetch failed: ${res.status}`);
  // Shape: { "user_id": { "model": { prompt_tokens, completion_tokens } } }
  return res.json() as Promise<
    Record<
      string,
      Record<string, { prompt_tokens: number; completion_tokens: number }>
    >
  >;
}

/** Fetch current limits for all known users from the limiter. */
export async function fetchAllLimits() {
  const res = await fetch(`${BASE}/admin/limits`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`Limits fetch failed: ${res.status}`);
  // Shape: { "user_id": { RPS, MaxTokens, MaxTokensPerReq, UsedTokens } }
  return res.json() as Promise<
    Record<
      string,
      {
        RPS: number;
        MaxTokens: number;
        MaxTokensPerReq: number;
        UsedTokens: number;
      }
    >
  >;
}

export async function setLimits(payload: SetLimitsPayload) {
  const res = await fetch(`${BASE}/admin/limits`, {
    method: "POST",
    headers: userHeaders(),
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error ?? `Error ${res.status}`);
  }
  return res.json();
}

export async function suspendUser(payload: SuspendPayload) {
  const res = await fetch(`${BASE}/admin/suspend`, {
    method: "POST",
    headers: userHeaders(),
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error ?? `Error ${res.status}`);
  }
  return res.json();
}
