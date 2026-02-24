// Thin auth helpers — API key is stored in sessionStorage to simulate a real session.
// In production, use HttpOnly cookies + a real token exchange.

const KEY_STORAGE = "proxy_api_key";
const USER_STORAGE = "proxy_user_id";
const ADMIN_STORAGE = "proxy_is_admin";

export function saveSession(userId: string, apiKey: string, isAdmin: boolean) {
  // TODO(Taman / critical / prod): Replace this with HttpOnly cookies + a real token exchange.
  // For demo purposes, we store the API key in sessionStorage.
  sessionStorage.setItem(KEY_STORAGE, apiKey);
  sessionStorage.setItem(USER_STORAGE, userId);
  sessionStorage.setItem(ADMIN_STORAGE, String(isAdmin));
}

export function getApiKey(): string | null {
  return sessionStorage.getItem(KEY_STORAGE);
}

export function getUserId(): string | null {
  return sessionStorage.getItem(USER_STORAGE);
}

export function isAdmin(): boolean {
  return sessionStorage.getItem(ADMIN_STORAGE) === "true";
}

export function clearSession() {
  sessionStorage.removeItem(KEY_STORAGE);
  sessionStorage.removeItem(USER_STORAGE);
  sessionStorage.removeItem(ADMIN_STORAGE);
}

export function isLoggedIn(): boolean {
  return !!getApiKey();
}

export async function login(
  username: string,
  password: string,
): Promise<{ user_id: string; api_key: string; is_admin: boolean }> {
  const res = await fetch("http://localhost:8000/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error ?? "Login failed");
  }
  return res.json();
}
