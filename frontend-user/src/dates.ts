const WEEKDAYS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];

export function parseYMD(s: string) {
  const [y, m, d] = s.split("-").map(Number);
  return new Date(y, (m || 1) - 1, d || 1);
}

export function fmt(d: Date) {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export function startOfWeek(d: Date) {
  const x = new Date(d);
  const wd = x.getDay() || 7;
  x.setDate(x.getDate() - (wd - 1));
  x.setHours(0, 0, 0, 0);
  return x;
}

export function addDays(d: Date, n: number) {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
}

export function weekDays(anchor: string) {
  const start = startOfWeek(parseYMD(anchor));
  return Array.from({ length: 7 }, (_, i) => {
    const d = addDays(start, i);
    return { date: fmt(d), label: WEEKDAYS[i], md: `${d.getMonth() + 1}/${d.getDate()}` };
  });
}

export function todayYMD() {
  return fmt(new Date());
}

export const SLOT_META = [
  { key: "breakfast" as const, label: "早餐" },
  { key: "lunch" as const, label: "午餐" },
  { key: "dinner" as const, label: "晚餐" },
];
