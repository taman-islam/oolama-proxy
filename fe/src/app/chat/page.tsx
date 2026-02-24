"use client";

import { useState, useRef, useEffect, FormEvent } from "react";
import { useRouter } from "next/navigation";
import { isLoggedIn, getUserId, getApiKey, clearSession } from "@/lib/auth";

interface Message {
  role: "user" | "assistant";
  content: string;
}

export default function ChatPage() {
  const router = useRouter();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [currentUser, setCurrentUser] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    setCurrentUser(getUserId());
  }, [router]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleLogout = () => {
    clearSession();
    router.replace("/login");
  };

  const send = async (e: FormEvent) => {
    e.preventDefault();
    const text = input.trim();
    if (!text || streaming) return;
    setInput("");

    const newMessages: Message[] = [
      ...messages,
      { role: "user", content: text },
    ];
    setMessages(newMessages);
    setStreaming(true);

    // Append an empty assistant turn we'll stream into
    setMessages((m) => [...m, { role: "assistant", content: "" }]);

    try {
      const res = await fetch("http://localhost:8000/v1/chat/completions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${getApiKey() ?? ""}`,
        },
        body: JSON.stringify({
          model: "llama3.2",
          stream: true,
          messages: newMessages.map((m) => ({
            role: m.role,
            content: m.content,
          })),
        }),
      });

      if (!res.ok || !res.body) {
        const err = await res.json().catch(() => ({ error: "Request failed" }));
        setMessages((m) => {
          const copy = [...m];
          copy[copy.length - 1] = {
            role: "assistant",
            content: `⚠️ ${(err as { error: string }).error}`,
          };
          return copy;
        });
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buf = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        const lines = buf.split("\n");
        buf = lines.pop() ?? "";
        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;
          const data = line.slice(6);
          if (data === "[DONE]") break;
          try {
            const chunk = JSON.parse(data);
            const delta: string = chunk.choices?.[0]?.delta?.content ?? "";
            if (delta) {
              setMessages((m) => {
                const copy = [...m];
                copy[copy.length - 1] = {
                  role: "assistant",
                  content: copy[copy.length - 1].content + delta,
                };
                return copy;
              });
            }
          } catch {
            /* skip malformed chunks */
          }
        }
      }
    } finally {
      setStreaming(false);
    }
  };

  return (
    <div className="flex flex-col h-screen" style={{ background: "var(--bg)" }}>
      {/* Header */}
      <header
        className="flex items-center justify-between px-6 py-3 border-b shrink-0"
        style={{ borderColor: "var(--border)", background: "var(--surface)" }}
      >
        <span
          className="font-semibold text-sm"
          style={{ color: "var(--purple)" }}
        >
          🔮 Proxy Chat
        </span>
        <div className="flex items-center gap-3">
          <button
            onClick={() => router.push("/usage")}
            className="text-xs px-3 py-1 rounded-lg transition-colors cursor-pointer"
            style={{
              color: "var(--purple-light)",
              background: "var(--surface)",
              border: "1px solid var(--border)",
            }}
          >
            📊 My Usage
          </button>

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
            className="text-xs px-3 py-1 rounded-lg"
            style={{
              background: "var(--bg)",
              color: "var(--muted)",
              border: "1px solid var(--border)",
            }}
          >
            Logout
          </button>
        </div>
      </header>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-6 flex flex-col gap-4">
        {messages.length === 0 && (
          <p
            className="text-center text-sm mt-16"
            style={{ color: "var(--muted)" }}
          >
            Start a conversation…
          </p>
        )}
        {messages.map((m, i) => (
          <div
            key={i}
            className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className="max-w-[75%] px-4 py-2 rounded-2xl text-sm whitespace-pre-wrap"
              style={
                m.role === "user"
                  ? {
                      background: "var(--purple)",
                      color: "#fff",
                      borderBottomRightRadius: "4px",
                    }
                  : {
                      background: "var(--surface)",
                      color: "var(--text)",
                      border: "1px solid var(--border)",
                      borderBottomLeftRadius: "4px",
                    }
              }
            >
              {m.content || <span style={{ opacity: 0.4 }}>▌</span>}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <form
        onSubmit={send}
        className="shrink-0 flex gap-2 px-4 py-4 border-t"
        style={{ borderColor: "var(--border)", background: "var(--surface)" }}
      >
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Message…"
          disabled={streaming}
          className="flex-1 px-4 py-2 rounded-xl text-sm outline-none"
          style={{
            background: "var(--bg)",
            border: "1px solid var(--border)",
            color: "var(--text)",
          }}
        />
        <button
          type="submit"
          disabled={streaming || !input.trim()}
          className="px-4 py-2 rounded-xl text-sm font-medium disabled:opacity-40 transition-opacity"
          style={{ background: "var(--purple)", color: "#fff" }}
        >
          {streaming ? "…" : "Send"}
        </button>
      </form>
    </div>
  );
}
