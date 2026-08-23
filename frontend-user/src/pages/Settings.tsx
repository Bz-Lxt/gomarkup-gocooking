import { useEffect, useState } from "react";
import { api } from "../api";
import type { Ingredient, Staple } from "../types";
import { ToastHost, useToast } from "../ui";

export default function Settings() {
  const toast = useToast();
  const [items, setItems] = useState<Staple[]>([]);
  const [ings, setIngs] = useState<Ingredient[]>([]);
  const [addId, setAddId] = useState(0);

  async function load() {
    try {
      const [s, i] = await Promise.all([api.staples(), api.ingredients()]);
      setItems(s);
      setIngs(i);
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "加载失败");
    }
  }
  useEffect(() => { void load(); }, []);

  return (
    <div>
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <h1 className="mb-2 font-display text-2xl">常备清单</h1>
      <p className="mb-4 max-w-3xl text-sm text-soot">
        过滤依据是这里的开关，不是「调料」分类。盐、生抽默认关闭采购；花椒、郫县豆瓣酱会正常出现在调料摊。
      </p>
      <ul className="divide-y divide-clay/70 rounded-2xl bg-white/40 shadow-ticket">
        {items.map((it) => (
          <li key={it.ingredient_id} className="flex items-center justify-between px-4 py-3">
            <div>
              <p className="font-medium">{it.name}</p>
              <p className="text-xs text-soot">{it.default_enabled ? "系统默认常备" : "自定义"}</p>
            </div>
            <label className="flex items-center gap-2 text-sm">
              当作常备（买菜时过滤）
              <input
                type="checkbox"
                className="h-5 w-5 accent-leaf"
                checked={it.enabled}
                onChange={async (e) => {
                  const next = items.map((x) => x.ingredient_id === it.ingredient_id ? { ...x, enabled: e.target.checked } : x);
                  setItems(next);
                  try {
                    await api.putStaples(next.map((x) => ({ ingredient_id: x.ingredient_id, enabled: x.enabled })));
                  } catch (err) {
                    toast.show(err instanceof Error ? err.message : "保存失败");
                    await load();
                  }
                }}
              />
            </label>
          </li>
        ))}
      </ul>
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <select value={addId} onChange={(e) => setAddId(Number(e.target.value))} className="rounded-xl border border-clay px-3 py-1.5">
          <option value={0}>选择要加入常备的食材</option>
          {ings.filter((i) => !items.some((s) => s.ingredient_id === i.id)).map((i) => (
            <option key={i.id} value={i.id}>{i.name}</option>
          ))}
        </select>
        <button
          type="button"
          className="rounded-full bg-leaf px-4 py-1.5 text-paper"
          onClick={async () => {
            if (!addId) {
              toast.show("请先选择食材");
              return;
            }
            await api.addStaple(addId, true);
            setAddId(0);
            await load();
          }}
        >
          加入常备
        </button>
      </div>
    </div>
  );
}
