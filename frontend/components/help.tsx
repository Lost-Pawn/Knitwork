"use client";
import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { Source } from "@/lib/types";
import { ErrorBox, Icon, TextContent, useUI } from "./ui";
type Message = {
  role: "user" | "assistant";
  text: string;
  sources?: Source[];
  mode?: string;
};
export default function Help() {
  const { t, locale } = useUI();
  const [messages, setMessages] = useState<Message[]>([]);
  const [question, setQuestion] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [end, setEnd] = useState<HTMLDivElement | null>(null);
  useEffect(
    () => end?.scrollIntoView({ behavior: "smooth", block: "nearest" }),
    [messages, busy, end],
  );
  return (
    <div className="help-page">
      <div className="page-intro">
        <div className="eyebrow">Knitwork</div>
        <h1>{t("help")}</h1>
        <p>{t("help_intro")}</p>
      </div>
      <div className="chat-shell">
        <div className="chat-header">
          <div className="chat-avatar">
            <Icon name="help" size={25} />
          </div>
          <div>
            <strong>{t("help")}</strong>
            <small>{t("local_retrieval")}</small>
          </div>
        </div>
        <div className="chat-messages" aria-live="polite">
          {!messages.length && (
            <div className="chat-welcome">
              <Icon name="help" size={42} />
              <h2>{t("help_intro")}</h2>
              <div className="suggestions">
                {["request", "training", "return", "extend"].map((k) => (
                  <button
                    key={k}
                    className="secondary"
                    onClick={() => setQuestion(t(k))}
                  >
                    {t(k)}
                  </button>
                ))}
              </div>
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} className={"message " + m.role}>
              <div className="bubble">
                <TextContent text={m.text} />
                {m.mode && <small className="mode">{t(m.mode)}</small>}
                {m.sources && m.sources.length > 0 && (
                  <details>
                    <summary>{t("sources")}</summary>
                    {m.sources.map((s, n) => (
                      <div key={s.id} className="source">
                        <strong>
                          [{n + 1}] {s.title}
                        </strong>
                        <TextContent text={s.body} />
                      </div>
                    ))}
                  </details>
                )}
              </div>
            </div>
          ))}
          {busy && (
            <div className="message assistant">
              <div className="bubble">{t("loading")}</div>
            </div>
          )}
          <div ref={setEnd} />
        </div>
        <ErrorBox error={error} />
        <form
          className="chat-input"
          onSubmit={async (e) => {
            e.preventDefault();
            if (busy || !question.trim()) return;
            const q = question;
            setQuestion("");
            setMessages((m) => [...m, { role: "user", text: q }]);
            setBusy(true);
            setError(null);
            try {
              const r = await api<{
                answer: string;
                sources: Source[];
                mode: string;
              }>("/help/chat", "POST", { question: q, locale });
              setMessages((m) => [
                ...m,
                {
                  role: "assistant",
                  text: r.answer,
                  sources: r.sources,
                  mode: r.mode,
                },
              ]);
            } catch (err) {
              setError(err);
            } finally {
              setBusy(false);
            }
          }}
        >
          <input
            aria-label={t("question")}
            placeholder={t("question")}
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            maxLength={1000}
            disabled={busy}
          />
          <button className="primary" disabled={busy || !question.trim()}>
            {t("send")}
            <Icon name="arrow" size={18} />
          </button>
        </form>
      </div>
    </div>
  );
}
