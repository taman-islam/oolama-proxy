// Types mirroring the Go backend response shapes

export interface ModelUsage {
  prompt_tokens: number;
  completion_tokens: number;
}

export interface LimitInfo {
  RPS: number;
  MaxTokens: number;
  MaxTokensPerReq: number;
  UsedTokens: number;
}

// GET /v1/usage  (as admin, we need all users — backend returns per-user)
export type UsageResponse = Record<string, Record<string, ModelUsage>>;

// GET /admin/limits equivalent — we derive this from the UI state
export type LimitsResponse = Record<string, LimitInfo>;

export interface SetLimitsPayload {
  user_id: string;
  rps: number;
  max_tokens: number;
  max_tokens_per_request: number;
}

export interface SuspendPayload {
  user_id: string;
}
