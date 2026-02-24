import { getApiKey } from "./auth";
import {
  AllUsageResponse,
  AllLimitsResponse,
  UsageResponse,
  SetLimitsRequest,
  SetLimitsResponse,
  SuspendUserRequest,
  SuspendUserResponse,
} from "../generated/api";

const BASE = "http://localhost:8000";

function userHeaders(): HeadersInit {
  const key = getApiKey() ?? "";
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${key}`,
  };
}

/** Fetch usage for the authenticated user. */
export async function fetchMyUsage(): Promise<UsageResponse> {
  const res = await fetch(`${BASE}/v1/usage`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`My usage fetch failed: ${res.status}`);
  const data = await res.json();
  return UsageResponse.fromJSON(data);
}

/** Fetch usage for ALL users via the admin endpoint. */
export async function fetchAllUsage(): Promise<AllUsageResponse> {
  const res = await fetch(`${BASE}/admin/usage`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`Usage fetch failed: ${res.status}`);
  const data = await res.json();
  return AllUsageResponse.fromJSON(data);
}

/** Fetch current limits for all known users from the limiter. */
export async function fetchAllLimits(): Promise<AllLimitsResponse> {
  const res = await fetch(`${BASE}/admin/limits`, { headers: userHeaders() });
  if (!res.ok) throw new Error(`Limits fetch failed: ${res.status}`);
  const data = await res.json();
  return AllLimitsResponse.fromJSON(data);
}

export async function setLimits(
  payload: SetLimitsRequest,
): Promise<SetLimitsResponse> {
  const reqBody = SetLimitsRequest.toJSON(SetLimitsRequest.create(payload));
  const res = await fetch(`${BASE}/admin/limits`, {
    method: "POST",
    headers: userHeaders(),
    body: JSON.stringify(reqBody),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error ?? `Error ${res.status}`);
  }
  const data = await res.json();
  return SetLimitsResponse.fromJSON(data);
}

export async function suspendUser(
  payload: SuspendUserRequest,
): Promise<SuspendUserResponse> {
  const reqBody = SuspendUserRequest.toJSON(SuspendUserRequest.create(payload));
  const res = await fetch(`${BASE}/admin/suspend`, {
    method: "POST",
    headers: userHeaders(),
    body: JSON.stringify(reqBody),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error ?? `Error ${res.status}`);
  }
  const data = await res.json();
  return SuspendUserResponse.fromJSON(data);
}
