/* eslint-disable @next/next/no-img-element */
"use client";

import { useState, useRef, useEffect, FormEvent } from "react";
import { useRouter } from "next/navigation";
import {
  isLoggedIn,
  getUserId,
  getApiKey,
  clearSession,
  isAdmin,
} from "@/lib/auth";
import { Navbar } from "@/components/Navbar";

interface Message {
  role: "user" | "assistant";
  content: string;
  image?: string | null;
}

export default function ChatPage() {
  const router = useRouter();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [selectedImage, setSelectedImage] = useState<string | null>(null);
  const [streaming, setStreaming] = useState(false);
  const [currentUser, setCurrentUser] = useState<string | null>(null);
  const [userIsAdmin, setUserIsAdmin] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    setCurrentUser(getUserId());
    setUserIsAdmin(isAdmin());

    // Load persisted chat
    const saved = localStorage.getItem(`chat_history_${getUserId()}`);
    if (saved) {
      try {
        setMessages(JSON.parse(saved));
      } catch {
        // ignore parse error
      }
    }
  }, [router]);

  useEffect(() => {
    if (messages.length > 0 && currentUser) {
      // 1. Limit raw message count (keep last 50 messages / 25 turns)
      let historyToSave = messages.slice(-50);

      try {
        let serialized = JSON.stringify(historyToSave);

        // 2. Limit byte size (localStorage quota is usually ~5MB, we cap at ~1MB to be safe)
        // If it's too large (likely due to base64 images), we progressively drop the oldest messages
        while (serialized.length > 1024 * 1024 && historyToSave.length > 2) {
          historyToSave = historyToSave.slice(2); // Drop oldest user/assistant pair
          serialized = JSON.stringify(historyToSave);
        }

        localStorage.setItem(`chat_history_${currentUser}`, serialized);
      } catch (err) {
        console.warn("Failed to serialize or save chat history", err);
      }
    }
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, currentUser]);

  useEffect(() => {
    if (!streaming) {
      inputRef.current?.focus();
    }
  }, [streaming]);

  const clearChat = () => {
    setMessages([]);
    localStorage.removeItem(`chat_history_${currentUser}`);
  };

  const handleLogout = () => {
    clearSession();
    router.replace("/login");
  };

  const handleImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (event) => {
      setSelectedImage(event.target?.result as string);
    };
    reader.readAsDataURL(file);
    // Reset input so the same file can be selected again
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const removeImage = () => {
    setSelectedImage(null);
  };

  const send = async (e: FormEvent) => {
    e.preventDefault();
    const text = input.trim();
    if ((!text && !selectedImage) || streaming) return;

    const currentInput = text;
    const currentImage = selectedImage;

    setInput("");
    setSelectedImage(null);

    const newMessages: Message[] = [
      ...messages,
      { role: "user", content: currentInput, image: currentImage },
    ];
    setMessages(newMessages);
    setStreaming(true);

    // Append an empty assistant turn we'll stream into
    setMessages((m) => [...m, { role: "assistant", content: "" }]);

    // Determine model and payload structure based on presence of image
    const model = currentImage ? "moondream" : "llama3.2";

    const apiMessages = newMessages.map((m) => {
      if (m.role === "user" && m.image) {
        return {
          role: m.role,
          content: [
            { type: "text", text: m.content || "Analyze this image." },
            { type: "image_url", image_url: { url: m.image } },
          ],
        };
      }
      return {
        role: m.role,
        content: m.content,
      };
    });

    try {
      const res = await fetch("http://localhost:8000/v1/chat/completions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${getApiKey() ?? ""}`,
        },
        body: JSON.stringify({
          model,
          stream: true,
          messages: apiMessages,
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
      <Navbar>
        <span className="text-xs px-2 py-1 rounded bg-black/20 text-gray-400 border border-white/5 ml-2">
          Auto-switches to {selectedImage ? "moondream" : "llama3.2"}
        </span>
        <button
          onClick={clearChat}
          className="text-xs px-3 py-1 rounded-lg transition-colors cursor-pointer"
          style={{
            color: "var(--red-text)",
            background: "var(--surface)",
            border: "1px solid var(--red-border)",
          }}
        >
          🗑️ Clear
        </button>
      </Navbar>

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
            className={`flex flex-col gap-2 ${m.role === "user" ? "items-end" : "items-start"}`}
          >
            {m.image && (
              <div
                className="max-w-[50%] rounded-xl overflow-hidden border"
                style={{ borderColor: "var(--border)" }}
              >
                <img
                  src={m.image}
                  alt="Uploaded attachment"
                  className="w-full object-cover"
                />
              </div>
            )}
            <div
              className={`max-w-[75%] px-4 py-2 text-sm whitespace-pre-wrap ${!m.content && m.image ? "hidden" : ""}`}
              style={
                m.role === "user"
                  ? {
                      background: "var(--purple)",
                      color: "#fff",
                      borderRadius: "16px",
                      borderBottomRightRadius: "4px",
                    }
                  : {
                      background: "var(--surface)",
                      color: "var(--text)",
                      border: "1px solid var(--border)",
                      borderRadius: "16px",
                      borderBottomLeftRadius: "4px",
                    }
              }
            >
              {m.content ||
                (m.role === "assistant" && (
                  <span style={{ opacity: 0.4 }}>▌</span>
                ))}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <form
        onSubmit={send}
        className="shrink-0 flex flex-col px-4 py-4 border-t"
        style={{ borderColor: "var(--border)", background: "var(--surface)" }}
      >
        {selectedImage && (
          <div className="mb-3 relative inline-block w-20 h-20 group">
            <img
              src={selectedImage}
              className="w-full h-full object-cover rounded-lg border"
              style={{ borderColor: "var(--border)" }}
              alt="Preview"
            />
            <button
              type="button"
              onClick={removeImage}
              className="absolute -top-2 -right-2 bg-red-500 hover:bg-red-600 text-white rounded-full w-5 h-5 flex items-center justify-center text-xs opacity-0 group-hover:opacity-100 transition-opacity transition-colors"
            >
              ×
            </button>
          </div>
        )}
        <div className="flex gap-2 relative">
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className="w-10 h-10 flex items-center justify-center rounded-xl transition-colors shrink-0"
            style={{
              background: "var(--bg)",
              border: "1px solid var(--border)",
              color: "var(--text)",
            }}
          >
            <span className="text-lg opacity-80">+</span>
          </button>
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleImageUpload}
            accept="image/*"
            className="hidden"
          />

          <input
            ref={inputRef}
            autoFocus
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={selectedImage ? "Ask about this image..." : "Message…"}
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
            disabled={streaming || (!input.trim() && !selectedImage)}
            className="px-4 py-2 rounded-xl text-sm font-medium disabled:opacity-40 transition-opacity"
            style={{ background: "var(--purple)", color: "#fff" }}
          >
            {streaming ? "…" : "Send"}
          </button>
        </div>
      </form>
    </div>
  );
}
