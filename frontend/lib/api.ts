export class APIError extends Error {
  constructor(
    public code: string,
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const res = await fetch("/api" + path, {
    method,
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const data = await res.json();
  if (!res.ok)
    throw new APIError(
      data.code || "server_error",
      data.error || "Request failed",
      res.status,
    );
  return data as T;
}
export function isStaff(role: string) {
  return ["owner", "manager", "admin"].includes(role);
}
export function canManage(role: string, id: number, owner: number) {
  return (
    ["manager", "admin"].includes(role) || (role === "owner" && id === owner)
  );
}
export function localISO(date: string, time: string) {
  return new Date(date + "T" + time + ":00+05:30").toISOString();
}
export function indianDate(value: string, locale = "en") {
  return new Intl.DateTimeFormat(locale + "-IN", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Kolkata",
  }).format(new Date(value));
}
export function nextDate() {
  const d = new Date(Date.now() + 86400000);
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Kolkata",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(d);
}
export function readPhoto(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    if (file.size > 2 * 1024 * 1024) {
      reject(new APIError("invalid_photo", "Maximum photo size is 2 MB", 400));
      return;
    }
    const r = new FileReader();
    r.onload = () => resolve(String(r.result));
    r.onerror = () => reject(new Error("Unable to read photo"));
    r.readAsDataURL(file);
  });
}
