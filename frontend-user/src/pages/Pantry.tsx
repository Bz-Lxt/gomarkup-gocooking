import { FormEvent, useEffect, useState } from "react";
import { api } from "../api";
import { todayYMD } from "../dates";
import type { Ingredient, Pantry as P } from "../types";
import { Dialog, FieldError, ToastHost, useToast } from "../ui";

const empty = { ingredient_id: 0, quantity: "", unit: "个", stocked_at: todayYMD(), expires_at: todayYMD() };

export default function Pantry() {
  const toast = useToast();
  const [rows, setRows] = useState<P[]>([]);
  const [ings, setIngs] = useState<Ingredient[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState(empty);
  const [errs, setErrs] = useState<Record<string, string>>({});
  const [del, setDel] = useState<P | null>(null);
  const [stall, setStall] = useState("全部");

  async function load() {
    try {
      const [p, i] = await Promise.all([api.pantry(), api.ingredients()]);
      setRows(p);
      setIngs(i);
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "加载失败");
    }
  }
  useEffect(() => { void load(); }, []);

  const stalls = ["全部", ...Array.from(new Set(rows.map((r) => r.stall).filter(Boolean)))];
  const shown = stall === "全部" ? rows : rows.filter((r) => r.stall === stall);

  function validate() {
    const e: Record<string, string> = {};
    if (!form.ingredient_id) e.ingredient_id = "请选择食材";
    if (!form.quantity || Number(form.quantity) <= 0) e.quantity = "数量必须大于 0";
    if (!form.unit.trim()) e.unit = "单位必填";
    if (!form.stocked_at) e.stocked_at = "请填写入库日";
    if (!form.expires_at) e.expires_at = "请填写保质期";
    if (form.stocked_at && form.expires_at && form.expires_at < form.stocked_at) e.expires_at = "保质期不得早于入库日";
    setErrs(e);
    return e;
  }

  async function onSave(ev: FormEvent) {
    ev.preventDefault();
    const e = validate();
    if (Object.keys(e).length) {
      toast.show("请先修正表单错误");
      return;
    }
    try {
      await api.addPantry({
        ingredient_id: form.ingredient_id,
        quantity: Number(form.quantity),
        unit: form.unit,
        stocked_at: form.stocked_at,
        expires_at: form.expires_at,
      });
      setOpen(false);
      setForm(empty);
      await load();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "保存失败");
    }
  }

  return (
    <div>
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <h1 className="font-display text-2xl">冰箱库存</h1>
        <select value={stall} onChange={(e) => setStall(e.target.value)} className="rounded-xl border border-clay px-3 py-1.5 text-sm">
          {stalls.map((s) => <option key={s}>{s}</option>)}
        </select>
        <button type="button" onClick={() => setOpen(true)} className="ml-auto rounded-full bg-terracotta px-4 py-1.5 text-paper">入库</button>
      </div>
      <div className="overflow-x-auto rounded-2xl bg-white/40 shadow-ticket">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="text-soot">
            <tr>
              <th className="px-4 py-3">食材</th>
              <th>数量</th>
              <th>摊位</th>
              <th>入库</th>
              <th>保质期</th>
              <th>状态</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {shown.map((r) => (
              <tr key={r.id} className="border-t border-clay/60">
                <td className="px-4 py-3 font-medium">{r.ingredient_name}</td>
                <td className="tabular-nums">{r.quantity} {r.unit}</td>
                <td>{r.stall}</td>
                <td>{r.stocked_at}</td>
                <td>{r.expires_at}</td>
                <td>
                  <span className={`rounded-full px-2 py-0.5 text-xs ${r.status === "expired" ? "bg-chili/15 text-chili" : r.status === "soon" ? "bg-yolk/25 text-ink" : "bg-leaf/15 text-leaf"}`}>
                    {r.status === "expired" ? "已过期" : r.status === "soon" ? `临期 ${r.days_left} 天` : "新鲜"}
                  </span>
                </td>
                <td>
                  <button type="button" className="text-chili" onClick={() => setDel(r)}>删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Dialog open={open} title="食材入库" onClose={() => setOpen(false)}>
        <form onSubmit={onSave} className="space-y-3">
          <label className="block text-sm">食材 *
            <select className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.ingredient_id} onChange={(e) => {
              const id = Number(e.target.value);
              const ing = ings.find((x) => x.id === id);
              setForm({ ...form, ingredient_id: id, unit: ing?.default_unit || form.unit });
            }}>
              <option value={0}>请选择</option>
              {ings.map((ing) => <option key={ing.id} value={ing.id}>{ing.name}</option>)}
            </select>
            <FieldError text={errs.ingredient_id} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="text-sm">数量 *
              <input className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.quantity} onChange={(e) => setForm({ ...form, quantity: e.target.value })} />
              <FieldError text={errs.quantity} />
            </label>
            <label className="text-sm">单位 *
              <input className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.unit} onChange={(e) => setForm({ ...form, unit: e.target.value })} />
              <FieldError text={errs.unit} />
            </label>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="text-sm">入库日 *
              <input type="date" className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.stocked_at} onChange={(e) => setForm({ ...form, stocked_at: e.target.value })} />
              <FieldError text={errs.stocked_at} />
            </label>
            <label className="text-sm">保质期 *
              <input type="date" className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} />
              <FieldError text={errs.expires_at} />
            </label>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setOpen(false)}>取消</button>
            <button type="submit" className="rounded-full bg-terracotta px-4 py-1.5 text-paper">保存</button>
          </div>
        </form>
      </Dialog>

      <Dialog open={!!del} title="删除这条库存？" onClose={() => setDel(null)}>
        <p className="text-sm text-soot">{del?.ingredient_name} 将从冰箱移除。</p>
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" onClick={() => setDel(null)}>取消</button>
          <button type="button" className="rounded-full bg-chili px-3 py-1 text-paper" onClick={async () => {
            if (!del) return;
            await api.deletePantry(del.id);
            setDel(null);
            await load();
          }}>确认删除</button>
        </div>
      </Dialog>
    </div>
  );
}
