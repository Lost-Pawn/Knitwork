"use client";
import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Operator, User } from "@/lib/types";
import {
  Empty,
  ErrorBox,
  Field,
  Icon,
  Modal,
  Status,
  TextContent,
  useUI,
} from "./ui";

export function Members({
  user,
  members,
  onChange,
}: {
  user: User;
  members: User[];
  onChange: () => Promise<void>;
}) {
  const { t, notify } = useUI();
  const [selected, setSelected] = useState<User | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const close = useCallback(() => {
    setSelected(null);
    setError(null);
  }, []);
  return (
    <>
      <div className="page-intro">
        <h1>{t("members")}</h1>
        <p>{t("review_needed")}</p>
      </div>
      <div className="people-grid">
        {members.map((m) => (
          <article className="person-card" key={m.id}>
            <div className="avatar">{m.name.slice(0, 1)}</div>
            <h3>{m.name}</h3>
            <p>{m.business}</p>
            <small>{m.email}</small>
            <Status
              value={
                m.suspended
                  ? "suspended"
                  : m.verified
                    ? "verified"
                    : "unverified"
              }
            />
            <p>{m.qualifications.join(", ") || t("none")}</p>
            <button className="secondary" onClick={() => setSelected(m)}>
              {t("review")}
            </button>
          </article>
        ))}
      </div>
      {selected && (
        <Modal title={selected.name} onClose={close}>
          <TextContent text={selected.verification_note || t("empty")} />
          <form
            className="form"
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError(null);
              const f = new FormData(e.currentTarget);
              const b: Record<string, unknown> = {
                verified: f.get("verified") === "on",
                qualifications: String(f.get("qualifications"))
                  .split(",")
                  .map((s) => s.trim())
                  .filter(Boolean),
                reason: f.get("reason"),
                allocation_group: f.get("allocation_group"),
              };
              if (user.role === "admin" && user.id !== selected.id) {
                b.role = f.get("role");
                b.suspended = f.get("suspended") === "on";
              }
              try {
                await api(`/members/${selected.id}`, "PATCH", b);
                await onChange();
                close();
                notify(t("saved"));
              } catch (err) {
                setError(err);
              } finally {
                setBusy(false);
              }
            }}
          >
            <label className="check-inline">
              <input
                type="checkbox"
                name="verified"
                defaultChecked={selected.verified}
              />
              {t("verified")}
            </label>
            <Field label={t("allocation_group")}>
              <input
                name="allocation_group"
                defaultValue={selected.allocation_group}
                maxLength={100}
              />
            </Field>
            <Field label={t("qualification_hint")}>
              <input
                name="qualifications"
                defaultValue={selected.qualifications.join(", ")}
              />
            </Field>
            {user.role === "admin" && user.id !== selected.id && (
              <>
                <Field label={t("profile")}>
                  <select name="role" defaultValue={selected.role}>
                    {["borrower", "owner", "manager", "admin"].map((k) => (
                      <option key={k} value={k}>
                        {t(k)}
                      </option>
                    ))}
                  </select>
                </Field>
                <label className="check-inline">
                  <input
                    type="checkbox"
                    name="suspended"
                    defaultChecked={selected.suspended}
                  />
                  {t("suspended")}
                </label>
              </>
            )}
            <Field label={t("reason")}>
              <textarea required minLength={5} name="reason" />
            </Field>
            <ErrorBox error={error} />
            <button className="primary" disabled={busy}>
              {t(busy ? "loading" : "save")}
            </button>
          </form>
        </Modal>
      )}
    </>
  );
}
export function Operators({
  operators,
  onChange,
  canEdit,
}: {
  operators: Operator[];
  onChange: () => Promise<void>;
  canEdit: boolean;
}) {
  const { t, notify } = useUI();
  const [selected, setSelected] = useState<Operator | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const close = useCallback(() => {
    setSelected(null);
    setError(null);
  }, []);
  return (
    <>
      <div className="section-title">
        <h1>{t("operators")}</h1>
        {canEdit && (
          <button
            className="primary"
            onClick={() =>
              setSelected({
                id: 0,
                name: "",
                qualifications: [],
                active: true,
                open_hour: 8,
                close_hour: 20,
              })
            }
          >
            <Icon name="plus" />
            {t("add_operator")}
          </button>
        )}
      </div>
      <div className="people-grid">
        {operators.map((o) => (
          <article className="person-card" key={o.id}>
            <div className="avatar">{o.name.slice(0, 1)}</div>
            <h3>{o.name}</h3>
            <Status value={o.active ? "active" : "suspended"} />
            <p>{o.qualifications.join(", ")}</p>
            <p>
              {o.open_hour}:00–{o.close_hour}:00 IST
            </p>
            {canEdit && (
              <button className="secondary" onClick={() => setSelected(o)}>
                {t("save")}
              </button>
            )}
          </article>
        ))}
      </div>
      {selected && (
        <Modal title={t("operator")} onClose={close}>
          <form
            className="form"
            onSubmit={async (e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              setBusy(true);
              try {
                await api("/operators", "POST", {
                  id: selected.id,
                  name: f.get("name"),
                  qualifications: String(f.get("qualifications"))
                    .split(",")
                    .map((s) => s.trim())
                    .filter(Boolean),
                  active: f.get("active") === "on",
                  open_hour: Number(f.get("open_hour")),
                  close_hour: Number(f.get("close_hour")),
                });
                await onChange();
                close();
                notify(t("saved"));
              } catch (err) {
                setError(err);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Field label={t("name")}>
              <input
                name="name"
                defaultValue={selected.name}
                required
                minLength={2}
              />
            </Field>
            <Field label={t("qualification_hint")}>
              <input
                name="qualifications"
                defaultValue={selected.qualifications.join(", ")}
              />
            </Field>
            <div className="form-grid">
              <Field label={t("start")}>
                <input
                  type="number"
                  name="open_hour"
                  defaultValue={selected.open_hour}
                  min={0}
                  max={23}
                  required
                />
              </Field>
              <Field label={t("end")}>
                <input
                  type="number"
                  name="close_hour"
                  defaultValue={selected.close_hour}
                  min={1}
                  max={24}
                  required
                />
              </Field>
            </div>
            <label className="check-inline">
              <input
                type="checkbox"
                name="active"
                defaultChecked={selected.active}
              />
              {t("active")}
            </label>
            <ErrorBox error={error} />
            <button className="primary" disabled={busy}>
              {t(busy ? "loading" : "save")}
            </button>
          </form>
        </Modal>
      )}
    </>
  );
}
export function Profile({
  user,
  onChange,
}: {
  user: User;
  onChange: () => Promise<void>;
}) {
  const { t, locale, notify } = useUI();
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  return (
    <div className="narrow">
      <div className="page-intro">
        <h1>{t("profile")}</h1>
        <p>
          {user.name} · {user.email}
        </p>
      </div>
      <div className="surface">
        <Status value={user.verified ? "verified" : "unverified"} />
        <p>
          {t("training")}: {user.qualifications.join(", ") || t("none")}
        </p>
        <form
          className="form"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            const f = new FormData(e.currentTarget);
            try {
              await api("/me", "PUT", {
                business: f.get("business"),
                verification_note: f.get("verification_note"),
                locale,
              });
              await onChange();
              notify(t("saved"));
              setError(null);
            } catch (err) {
              setError(err);
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label={t("business")}>
            <input
              name="business"
              defaultValue={user.business}
              maxLength={150}
            />
          </Field>
          <Field label={t("verification_note")}>
            <textarea
              name="verification_note"
              defaultValue={user.verification_note}
              maxLength={1000}
            />
          </Field>
          <ErrorBox error={error} />
          <button className="primary" disabled={busy}>
            {t(busy ? "loading" : "save")}
          </button>
        </form>
      </div>
    </div>
  );
}
type Stat = {
  id: number;
  name: string;
  maintenance: boolean;
  requests: number;
  waiting: number;
  overdue: number;
  completed: number;
  hours: number;
};
export function Dashboard() {
  const { t } = useUI();
  const [data, setData] = useState<Stat[]>([]);
  const [error, setError] = useState<unknown>(null);
  useEffect(() => {
    api<Stat[]>("/dashboard").then(setData).catch(setError);
  }, []);
  return (
    <>
      <div className="page-intro">
        <h1>{t("centre")}</h1>
        <p>{t("shared_centre")}</p>
      </div>
      <div className="stat-grid">
        {["requests", "waiting", "overdue", "completed"].map((k, i) => (
          <div className="stat-card" key={k}>
            <Icon name={i === 2 ? "clock" : "bookings"} />
            <strong>
              {data.reduce((n, d) => n + Number(d[k as keyof Stat]), 0)}
            </strong>
            <span>
              {t(["requested", "waitlisted", "overdue", "completed"][i])}
            </span>
          </div>
        ))}
      </div>
      <div className="surface table-wrap">
        <table>
          <thead>
            <tr>
              <th>{t("name")}</th>
              <th>{t("completed")}</th>
              <th>{t("usage")}</th>
              <th>{t("available")}</th>
            </tr>
          </thead>
          <tbody>
            {data.map((d) => (
              <tr key={d.id}>
                <td>{d.name}</td>
                <td>{d.completed}</td>
                <td>{Number(d.hours).toFixed(1)}</td>
                <td>
                  <Status value={d.maintenance ? "maintenance" : "available"} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ErrorBox error={error} />
      {!data.length && <Empty />}
    </>
  );
}
