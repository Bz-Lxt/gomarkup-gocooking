import { useEffect, useMemo, useState } from "react";
import { api } from "../api";
import { fmt, startOfWeek, addDays, todayYMD } from "../dates";
import type { ListItem, Shopping as ShoppingT } from "../types";
import { ToastHost, useToast } from "../ui";

export default function Shopping() {
  const toast = useToast();
  const [from, setFrom] = useState(() => fmt(startOfWeek(new Date())));
  const [to, setTo] = useState(() => fmt(addDays(startOfWeek(new Date()), 6)));
  const [data, setData] = useState<ShoppingT | null>(null);
  const [mode, setMode] = useState<"stall" | "produce">("stall");
  const [openFiltered, setOpenFiltered] = useState(false);
  const [openId, setOpenId] = useState<string>("");

  async function gen() {
    try {
      setData(await api.generate(from, to));
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "生成失败");
    }
  }
  useEffect(() => {
    void gen();
  }, []);

  const groups = mode === "stall" ? data?.groups_by_stall : data?.groups_by_produce;

  async function toggle(it: ListItem, checked: boolean) {
    try {
      await api.check({ from, to, ingredient_id: it.ingredient_id, unit: it.check_unit || it.unit, dimension: it.dimension, checked });
      await gen();
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "勾选失败");
    }
  }

  const text = useMemo(() => {
    if (!data) return "";
    const lines = [`买菜清单 ${data.from} ~ ${data.to}`, ""];
    for (const g of data.groups_by_stall) {
      lines.push(`【${g.key}】`);
      for (const it of g.items) lines.push(`- [${it.checked ? "x" : " "}] ${it.display}`);
      lines.push("");
    }
    if (data.filtered.length) {
      lines.push(`已过滤常备 ${data.filtered.length} 项`);
    }
    return lines.join("\n");
  }, [data]);

  return (
    <div>
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <h1 className="font-display text-2xl">买菜清单</h1>
        <label className="text-sm">从
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} className="ml-2 rounded-lg border border-clay px-2 py-1" />
        </label>
        <label className="text-sm">到
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} className="ml-2 rounded-lg border border-clay px-2 py-1" />
        </label>
        <button type="button" onClick={() => void gen()} className="rounded-full bg-terracotta px-4 py-1.5 text-paper">重新生成</button>
        <button
          type="button"
          className="rounded-full border border-clay px-3 py-1.5 text-sm"
          onClick={async () => {
            await navigator.clipboard.writeText(text);
            toast.show("已复制为纯文本");
          }}
        >
          复制文本
        </button>
        <button
          type="button"
          className="rounded-full border border-clay px-3 py-1.5 text-sm"
          onClick={() => {
            const blob = new Blob([`# 买菜清单\n\n${text}`], { type: "text/markdown" });
            const a = document.createElement("a");
            a.href = URL.createObjectURL(blob);
            a.download = `shopping-${todayYMD()}.md`;
            a.click();
          }}
        >
          下载 Markdown
        </button>
      </div>

      {data?.expiry_alerts?.length ? (
        <div className="mb-3 rounded-xl bg-yolk/20 px-4 py-3 text-sm">
          {data.expiry_alerts.map((a) => <p key={a.ingredient_id}>{a.message}</p>)}
        </div>
      ) : null}

      <div className="mb-4 flex gap-2">
        <button type="button" onClick={() => setMode("stall")} className={`rounded-full px-3 py-1 text-sm ${mode === "stall" ? "bg-ink text-paper" : "bg-clay"}`}>菜市场摊位</button>
        <button type="button" onClick={() => setMode("produce")} className={`rounded-full px-3 py-1 text-sm ${mode === "produce" ? "bg-ink text-paper" : "bg-clay"}`}>蔬菜分类</button>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {(groups || []).map((g) => (
          <section key={g.key} className="rounded-2xl bg-white/40 p-4 shadow-ticket">
            <h2 className="mb-3 font-display text-lg">{g.key}</h2>
            <ul className="space-y-2">
              {g.items.map((it) => {
                const kid = `${it.ingredient_id}-${it.dimension}-${it.unit}`;
                return (
                  <li key={kid} className="rounded-xl bg-paper/80 px-3 py-2">
                    <label className="flex items-start gap-3">
                      <input
                        type="checkbox"
                        className="mt-1 h-[22px] w-[22px] accent-leaf"
                        checked={it.checked}
                        onChange={(e) => void toggle(it, e.target.checked)}
                      />
                      <div className="flex-1">
                        <p className={`font-medium ${it.checked ? "text-leaf line-through" : ""}`}>{it.display}</p>
                        {it.needs_review ? <p className="text-xs text-yolk">同名不同单位，未强行合并</p> : null}
                        <button type="button" className="text-xs text-terracotta" onClick={() => setOpenId(openId === kid ? "" : kid)}>
                          来自哪些菜
                        </button>
                        {openId === kid ? (
                          <ul className="mt-1 text-xs text-soot">
                            {it.sources.map((s, i) => (
                              <li key={i}>{s.date} {s.slot_label} · {s.recipe_name} {s.quantity} {s.unit}</li>
                            ))}
                          </ul>
                        ) : null}
                      </div>
                    </label>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
      </div>

      <div className="mt-6 rounded-2xl border border-dashed border-clay p-4">
        <button type="button" className="font-display" onClick={() => setOpenFiltered((v) => !v)}>
          已过滤 {data?.filtered.length || 0} 项常备调料 {openFiltered ? "▾" : "▸"}
        </button>
        {openFiltered ? (
          <ul className="mt-3 space-y-2">
            {(data?.filtered || []).map((it) => (
              <li key={`${it.ingredient_id}-${it.unit}`} className="flex items-center justify-between text-sm">
                <span>{it.display}</span>
                <button
                  type="button"
                  className="rounded-full bg-leaf px-3 py-1 text-paper"
                  onClick={async () => {
                    await api.restore({ from, to, ingredient_id: it.ingredient_id, unit: it.check_unit || it.unit, dimension: it.dimension });
                    await gen();
                  }}
                >
                  盐用完了，加回清单
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  );
}
