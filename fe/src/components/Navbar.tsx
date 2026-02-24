"use client";

import { useRouter, usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { getUserId, isAdmin, clearSession } from "@/lib/auth";

export function Navbar({ children }: { children?: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [currentUser, setCurrentUser] = useState<string | null>(null);
  const [userIsAdmin, setUserIsAdmin] = useState(false);

  useEffect(() => {
    setCurrentUser(getUserId());
    setUserIsAdmin(isAdmin());
  }, []);

  const handleLogout = () => {
    clearSession();
    router.replace("/login");
  };

  const NavLink = ({ href, label }: { href: string; label: string }) => {
    const active = pathname === href;
    return (
      <button
        onClick={() => router.push(href)}
        className="text-xs px-4 py-1.5 rounded-lg transition-colors font-semibold"
        style={{
          background: active ? "var(--purple)" : "transparent",
          color: active ? "#fff" : "var(--muted)",
        }}
      >
        {label}
      </button>
    );
  };

  return (
    <header
      className="flex items-center justify-between px-6 py-3 border-b shrink-0 h-14"
      style={{ borderColor: "var(--border)", background: "var(--surface)" }}
    >
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-2">
          <span className="text-xl">🔮</span>
          <span
            className="font-bold text-sm"
            style={{ color: "var(--purple)" }}
          >
            Proxy
          </span>
        </div>

        <div
          className="flex items-center gap-1 border-l pl-6"
          style={{ borderColor: "var(--border)" }}
        >
          <NavLink href="/" label="Chat" />
          <NavLink href="/usage" label="Usage" />
          {userIsAdmin && <NavLink href="/admin" label="Admin" />}
        </div>
      </div>

      <div className="flex items-center gap-4">
        {children && (
          <div
            className="flex items-center gap-3 pr-4 border-r"
            style={{ borderColor: "var(--border)" }}
          >
            {children}
          </div>
        )}

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
            className="text-xs px-3 py-1.5 rounded-lg transition-colors"
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
    </header>
  );
}
