"use client";
import { useCallback, useEffect, useState } from "react";
import { api, APIError, indianDate, isStaff } from "@/lib/api";
import { Booking, Notice, Operator, Resource, User } from "@/lib/types";
import { languages, Locale, translate } from "@/lib/i18n";
import {
  Empty,
  ErrorBox,
  Field,
  Icon,
  Machine,
  TextContent,
  UIContext,
} from "@/components/ui";
import Resources from "@/components/resources";
import Bookings from "@/components/bookings";
import {
  Dashboard,
  Members,
  Operators,
  Profile,
} from "@/components/management";
import Help from "@/components/help";

export default function Home() {
  const [locale, setLocale] = useState<Locale>("en");
  const [user, setUser] = useState<User | null>(null);
  const [resources, setResources] = useState<Resource[]>([]);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [operators, setOperators] = useState<Operator[]>([]);
  const [members, setMembers] = useState<User[]>([]);
  const [notices, setNotices] = useState<Notice[]>([]);
  const [page, setPage] = useState("explore");
  const [initialID, setInitialID] = useState<number>();
  const [config, setConfig] = useState({ demo: false, translation: false });
  const [loading, setLoading] = useState(true);
  const [register, setRegister] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [toast, setToast] = useState("");
  const t = useCallback((key: string) => translate(locale, key), [locale]);
  const refresh = useCallback(async () => {
    const me = await api<User>("/me");
    setUser(me);
    const [r, b, n] = await Promise.all([
      api<Resource[]>("/resources"),
      api<Booking[]>("/bookings"),
      api<Notice[]>("/notifications"),
    ]);
    setResources(r);
    setBookings(b);
    setNotices(n);
    if (isStaff(me.role)) setOperators(await api<Operator[]>("/operators"));
    if (["manager", "admin"].includes(me.role))
      setMembers(await api<User[]>("/members"));
  }, []);
  useEffect(() => {
    const saved = localStorage.getItem("knitwork-language");
    if (saved && saved in languages) setLocale(saved as Locale);
    api<typeof config>("/config")
      .then(setConfig)
      .catch(() => {});
    refresh()
      .catch((e) => {
        if (!(e instanceof APIError && e.status === 401)) setError(e);
      })
      .finally(() => setLoading(false));
  }, [refresh]);
  useEffect(() => {
    document.documentElement.lang = locale;
    localStorage.setItem("knitwork-language", locale);
  }, [locale]);
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(""), 4500);
    return () => clearTimeout(timer);
  }, [toast]);
  useEffect(() => {
    if (!user) return;
    const timer = setInterval(
      () =>
        api<Notice[]>("/notifications")
          .then(setNotices)
          .catch(() => {}),
      30000,
    );
    return () => clearInterval(timer);
  }, [user]);
  async function login(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    setBusy(true);
    setError(null);
    try {
      await api(
        register ? "/auth/register" : "/auth/login",
        "POST",
        register
          ? {
              name: f.get("name"),
              email: f.get("email"),
              password: f.get("password"),
              business: f.get("business"),
            }
          : { email: f.get("email"), password: f.get("password") },
      );
      await refresh();
      setPage("explore");
    } catch (err) {
      setError(err);
    } finally {
      setBusy(false);
    }
  }
  const language = (
    <label className="language">
      <span className="sr-only">{t("language")}</span>
      <span aria-hidden="true">अ</span>
      <select
        value={locale}
        onChange={(e) => setLocale(e.target.value as Locale)}
        aria-label={t("language")}
      >
        {Object.entries(languages).map(([code, label]) => (
          <option key={code} value={code}>
            {label}
          </option>
        ))}
      </select>
    </label>
  );
  const navigate = (p: string) => {
    setPage(p);
    setInitialID(undefined);
    setError(null);
  };
  return (
    <UIContext.Provider
      value={{ locale, t, translation: config.translation, notify: setToast }}
    >
      {loading ? (
        <div className="loading-screen">
          <div className="brand-mark">k</div>
          <p>{t("loading")}</p>
        </div>
      ) : !user ? (
        <div className="auth-page">
          <section className="auth-story">
            <div className="brand">
              <div className="brand-mark">k</div>knitwork<span>®</span>
            </div>
            <div className="auth-copy">
              <span className="eyebrow">{t("shared_centre")}</span>
              <h1>{t("login_intro")}</h1>
              <p>{t("intro")}</p>
              <div className="auth-machine">
                <Machine category="embroidery" />
              </div>
              <div className="auth-footnote">
                <Icon name="check" />
                {t("review_needed")}
              </div>
            </div>
          </section>
          <section className="auth-form-side">
            <div className="auth-top">{language}</div>
            <div className="auth-form">
              <h2>{t(register ? "register" : "login")}</h2>
              <p className="muted">{t("shared_centre")}</p>
              <form onSubmit={login} className="form">
                {register && (
                  <>
                    <Field label={t("name")}>
                      <input
                        name="name"
                        autoComplete="name"
                        required
                        minLength={2}
                      />
                    </Field>
                    <Field label={t("business")}>
                      <input name="business" required maxLength={150} />
                    </Field>
                  </>
                )}
                <Field label={t("email")}>
                  <input
                    name="email"
                    type="email"
                    autoComplete="username"
                    required
                  />
                </Field>
                <Field label={t("password")}>
                  <input
                    name="password"
                    type="password"
                    autoComplete={
                      register ? "new-password" : "current-password"
                    }
                    minLength={register ? 10 : undefined}
                    required
                  />
                  {register && <small>{t("password_hint")}</small>}
                </Field>
                <ErrorBox error={error} />
                <button className="primary full" disabled={busy}>
                  {t(busy ? "loading" : register ? "register" : "login")}
                  <Icon name="arrow" />
                </button>
              </form>
              <button
                className="text-button auth-toggle"
                onClick={() => {
                  setRegister(!register);
                  setError(null);
                }}
              >
                {t(register ? "login" : "register")}
              </button>
              {config.demo && (
                <div className="demo-box">
                  <small>{t("demo")}</small>
                  <div className="demo-buttons">
                    {[
                      ["borrower", "asha"],
                      ["manager", "manager"],
                      ["admin", "admin"],
                    ].map(([role, account]) => (
                      <button
                        key={role}
                        className="secondary"
                        disabled={busy}
                        onClick={async () => {
                          setBusy(true);
                          setError(null);
                          try {
                            await api("/auth/login", "POST", {
                              email: account + "@knitwork.test",
                              password: "KnitworkDemo2026!",
                            });
                            await refresh();
                          } catch (err) {
                            setError(err);
                          } finally {
                            setBusy(false);
                          }
                        }}
                      >
                        {t(role)}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </section>
        </div>
      ) : (
        <div className="app-shell">
          <aside className="sidebar">
            <button className="brand" onClick={() => navigate("explore")}>
              <div className="brand-mark">k</div>knitwork<span>®</span>
            </button>
            <div className="workspace-label">{t("shared_centre")}</div>
            <nav aria-label="Knitwork">
              {[
                "explore",
                "bookings",
                ...(isStaff(user.role) ? ["centre", "operators"] : []),
                ...(["manager", "admin"].includes(user.role)
                  ? ["members"]
                  : []),
                "help",
                "profile",
              ].map((p) => (
                <button
                  key={p}
                  className={page === p ? "nav-item active" : "nav-item"}
                  onClick={() => navigate(p)}
                >
                  <Icon name={p} />
                  <span>{t(p)}</span>
                  {p === "bookings" &&
                    bookings.filter((b) => b.status === "requested").length >
                      0 && (
                      <small>
                        {
                          bookings.filter((b) => b.status === "requested")
                            .length
                        }
                      </small>
                    )}
                </button>
              ))}
            </nav>
            <div className="sidebar-bottom">
              <div className="member-chip">
                <div className="avatar">{user.name[0]}</div>
                <div>
                  <strong>{user.name}</strong>
                  <small>{t(user.role)}</small>
                </div>
              </div>
              <button
                className="nav-item"
                onClick={async () => {
                  try {
                    await api("/auth/logout", "POST", {});
                    setUser(null);
                    setBookings([]);
                    setResources([]);
                    setMembers([]);
                    setError(null);
                  } catch (err) {
                    setError(err);
                  }
                }}
              >
                <Icon name="logout" />
                {t("logout")}
              </button>
            </div>
          </aside>
          <div className="workspace">
            <header className="topbar">
              <div className="breadcrumb">
                Knitwork <span>/</span>
                <strong>{t(page)}</strong>
              </div>
              <div className="topbar-actions">
                {language}
                <button
                  className="icon-button notification-button"
                  aria-label={t("notifications")}
                  onClick={() => navigate("notifications")}
                >
                  <Icon name="notifications" />
                  {notices.some((n) => !n.read) && <i />}
                </button>
                <div className="avatar small">{user.name[0]}</div>
              </div>
            </header>
            <main id="main">
              <ErrorBox error={error} />
              {page === "explore" && (
                <Resources
                  user={user}
                  resources={resources}
                  members={members}
                  onChange={refresh}
                  onBooked={() => navigate("bookings")}
                />
              )}{" "}
              {page === "bookings" && (
                <Bookings
                  user={user}
                  bookings={bookings}
                  operators={operators}
                  onChange={refresh}
                  initialID={initialID}
                />
              )}{" "}
              {page === "centre" && <Dashboard />}
              {page === "members" && (
                <Members user={user} members={members} onChange={refresh} />
              )}{" "}
              {page === "operators" && (
                <Operators
                  operators={operators}
                  onChange={refresh}
                  canEdit={["manager", "admin"].includes(user.role)}
                />
              )}{" "}
              {page === "profile" && <Profile user={user} onChange={refresh} />}{" "}
              {page === "help" && <Help />}
              {page === "notifications" && (
                <>
                  <div className="section-title">
                    <h1>{t("notifications")}</h1>
                    <button
                      className="secondary"
                      onClick={async () => {
                        await api("/notifications/read", "POST", {});
                        await refresh();
                      }}
                    >
                      {t("mark_read")}
                    </button>
                  </div>
                  <div className="notice-list">
                    {notices.map((n) => (
                      <article
                        key={n.id}
                        className={"notice-item " + (n.read ? "read" : "")}
                      >
                        <Icon name="notifications" />
                        <div>
                          <strong>{t(n.kind)}</strong>
                          <p>{t("notice_" + n.kind)}</p>
                          <small>{indianDate(n.created_at, locale)}</small>
                          {n.booking_id && (
                            <button
                              className="text-button"
                              onClick={() => {
                                setInitialID(n.booking_id!);
                                setPage("bookings");
                              }}
                            >
                              {t("booking")} #{n.booking_id}
                              <Icon name="arrow" size={15} />
                            </button>
                          )}
                        </div>
                      </article>
                    ))}
                  </div>
                  {!notices.length && <Empty />}
                </>
              )}
            </main>
            <footer>
              Knitwork <span>·</span> {t("timezone")}
            </footer>
          </div>
        </div>
      )}
      {toast && (
        <div className="toast" role="status">
          <Icon name="check" />
          {toast}
        </div>
      )}
    </UIContext.Provider>
  );
}
