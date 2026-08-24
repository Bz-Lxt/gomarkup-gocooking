const TOKEN_KEY = "gocooking_token";

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}
export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

type Err = { error?: { code?: string; message?: string; details?: { field: string; message: string }[] } };

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { ...(init.headers as Record<string, string>) };
  if (init.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
  const tok = getToken();
  if (tok) headers.Authorization = `Bearer ${tok}`;
  const res = await fetch(path, { ...init, headers });
  if (res.status === 204) return undefined as T;
  const body = (await res.json().catch(() => ({}))) as Err & { data?: T; meta?: unknown };
  if (!res.ok) {
    const msg = body.error?.message || "请求失败";
    const extra = body.error?.details?.map((d) => `${d.field}: ${d.message}`).join("；");
    throw new Error(extra ? `${msg}（${extra}）` : msg);
  }
  return (body.data as T) ?? (body as T);
}

async function list<T>(path: string): Promise<T[]> {
  const headers: Record<string, string> = {};
  const tok = getToken();
  if (tok) headers.Authorization = `Bearer ${tok}`;
  const res = await fetch(path, { headers });
  const body = await res.json();
  if (!res.ok) throw new Error(body.error?.message || "加载失败");
  return (body.data || []) as T[];
}

export const api = {
  login: (username: string, password: string) =>
    req<{ token: string; username: string }>("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  recipes: (q = "") => list<import("./types").Recipe>("/api/v1/recipes?per_page=80" + (q ? `&q=${encodeURIComponent(q)}` : "")),
  ingredients: (q = "") =>
    list<import("./types").Ingredient>("/api/v1/ingredients?per_page=200" + (q ? `&q=${encodeURIComponent(q)}` : "")),
  createRecipe: (body: unknown) => req("/api/v1/recipes", { method: "POST", body: JSON.stringify(body) }),
  updateRecipe: (id: number, body: unknown) => req(`/api/v1/recipes/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteRecipe: (id: number) => req(`/api/v1/recipes/${id}`, { method: "DELETE" }),
  duplicateRecipe: (id: number) => req(`/api/v1/recipes/${id}/duplicate`, { method: "POST" }),
  week: (week: string) => req<import("./types").WeekPlan>(`/api/v1/meal-plan?week=${week}`),
  addSlot: (body: unknown) => req("/api/v1/meal-plan/slots", { method: "POST", body: JSON.stringify(body) }),
  patchSlot: (id: number, body: unknown) => req(`/api/v1/meal-plan/slots/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  deleteSlot: (id: number) => req(`/api/v1/meal-plan/slots/${id}`, { method: "DELETE" }),
  clearWeek: (week: string) => req("/api/v1/meal-plan/clear", { method: "POST", body: JSON.stringify({ week }) }),
  copyNext: (week: string) => req("/api/v1/meal-plan/copy-next", { method: "POST", body: JSON.stringify({ week }) }),
  pantry: () => req<import("./types").Pantry[]>("/api/v1/pantry"),
  addPantry: (body: unknown) => req("/api/v1/pantry", { method: "POST", body: JSON.stringify(body) }),
  updatePantry: (id: number, body: unknown) => req(`/api/v1/pantry/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deletePantry: (id: number) => req(`/api/v1/pantry/${id}`, { method: "DELETE" }),
  generate: (from: string, to: string) =>
    req<import("./types").Shopping>("/api/v1/shopping-lists/generate", { method: "POST", body: JSON.stringify({ from, to }) }),
  check: (body: unknown) => req("/api/v1/shopping-lists/checks", { method: "PATCH", body: JSON.stringify(body) }),
  restore: (body: unknown) => req("/api/v1/shopping-lists/restore", { method: "POST", body: JSON.stringify(body) }),
  staples: () => req<import("./types").Staple[]>("/api/v1/settings/staples"),
  putStaples: (items: { ingredient_id: number; enabled: boolean }[]) =>
    req("/api/v1/settings/staples", { method: "PUT", body: JSON.stringify({ items }) }),
  addStaple: (ingredient_id: number, enabled: boolean) =>
    req("/api/v1/settings/staples", { method: "POST", body: JSON.stringify({ ingredient_id, enabled }) }),
};
