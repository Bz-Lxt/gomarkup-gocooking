import { FormEvent, useEffect, useState } from "react";
import { api } from "../api";
import type { Ingredient, Recipe } from "../types";
import { Dialog, FieldError, ToastHost, useToast } from "../ui";

type ItemDraft = { ingredient_id: number; quantity: string; unit: string; optional: boolean };

const empty = { name: "", cuisine_tag: "家常", servings: "2", steps: "", items: [{ ingredient_id: 0, quantity: "1", unit: "个", optional: false }] as ItemDraft[] };

export default function Recipes() {
  const toast = useToast();
  const [rows, setRows] = useState<Recipe[]>([]);
  const [ings, setIngs] = useState<Ingredient[]>([]);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Recipe | null>(null);
  const [form, setForm] = useState(empty);
  const [errs, setErrs] = useState<Record<string, string>>({});
  const [del, setDel] = useState<Recipe | null>(null);

  async function load() {
    try {
      const [r, i] = await Promise.all([api.recipes(q), api.ingredients()]);
      setRows(r);
      setIngs(i);
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "加载失败");
    }
  }
  useEffect(() => { void load(); }, [q]);

  function startCreate() {
    setEditing(null);
    setForm(empty);
    setErrs({});
    setOpen(true);
  }
  function startEdit(r: Recipe) {
    setEditing(r);
    setForm({
      name: r.name,
      cuisine_tag: r.cuisine_tag,
      servings: String(r.servings),
      steps: (r.steps || []).join("\n"),
      items: (r.items || []).map((it) => ({
        ingredient_id: it.ingredient_id, quantity: String(it.quantity), unit: it.unit, optional: it.optional,
      })),
    });
    setErrs({});
    setOpen(true);
  }

  function validate() {
    const e: Record<string, string> = {};
    if (!form.name.trim()) e.name = "请填写菜名";
    const serv = Number(form.servings);
    if (!serv || serv < 1 || serv > 20) e.servings = "份数须在 1–20";
    if (!form.items.length) e.items = "至少一味食材";
    form.items.forEach((it, i) => {
      if (!it.ingredient_id) e[`items.${i}.ingredient_id`] = "请选择食材";
      if (!it.unit.trim()) e[`items.${i}.unit`] = "单位必填";
      if (Number(it.quantity) < 0) e[`items.${i}.quantity`] = "数量不能为负";
    });
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
    const body = {
      name: form.name.trim(),
      cuisine_tag: form.cuisine_tag.trim(),
      servings: Number(form.servings),
      steps: form.steps.split("\n").map((s) => s.trim()).filter(Boolean),
      items: form.items.map((it) => ({
        ingredient_id: it.ingredient_id, quantity: Number(it.quantity), unit: it.unit, optional: it.optional,
      })),
    };
    try {
      if (editing && editing.user_id) await api.updateRecipe(editing.id, body);
      else if (editing) {
        toast.show("系统菜谱请先复制为副本再改");
        return;
      } else await api.createRecipe(body);
      setOpen(false);
      await load();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "保存失败");
    }
  }

  return (
    <div>
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <h1 className="font-display text-2xl">私房菜谱</h1>
        <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索菜名" className="rounded-xl border border-clay px-3 py-1.5 text-sm" />
        <button type="button" onClick={startCreate} className="ml-auto rounded-full bg-terracotta px-4 py-1.5 text-paper">新建菜谱</button>
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        {rows.map((r) => (
          <article key={r.id} className="rounded-2xl bg-white/40 p-4 shadow-ticket">
            <div className="flex items-start justify-between gap-2">
              <div>
                <h2 className="font-display text-lg">{r.name}</h2>
                <p className="text-xs text-soot">{r.cuisine_tag} · {r.servings} 人份 {r.user_id ? "" : "· 系统"}</p>
              </div>
              <div className="flex gap-2 text-xs">
                <button type="button" className="text-terracotta" onClick={() => startEdit(r)}>编辑</button>
                <button type="button" className="text-leaf" onClick={async () => { await api.duplicateRecipe(r.id); await load(); }}>复制</button>
                {r.user_id ? <button type="button" className="text-chili" onClick={() => setDel(r)}>删除</button> : null}
              </div>
            </div>
            <ul className="mt-2 text-sm text-soot">
              {(r.items || []).slice(0, 4).map((it) => (
                <li key={it.id}>{it.ingredient_name} {it.quantity} {it.unit}</li>
              ))}
            </ul>
          </article>
        ))}
      </div>

      <Dialog open={open} title={editing ? "编辑菜谱" : "新建菜谱"} onClose={() => setOpen(false)}>
        <form onSubmit={onSave} className="max-h-[70vh] space-y-3 overflow-auto pr-1">
          <label className="block text-sm">菜名 *
            <input className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            <FieldError text={errs.name} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="text-sm">菜系
              <input className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.cuisine_tag} onChange={(e) => setForm({ ...form, cuisine_tag: e.target.value })} />
            </label>
            <label className="text-sm">默认份数 *
              <input className="mt-1 w-full rounded-lg border border-clay px-2 py-1.5" value={form.servings} onChange={(e) => setForm({ ...form, servings: e.target.value })} />
              <FieldError text={errs.servings} />
            </label>
          </div>
          <label className="block text-sm">步骤（一行一步）
            <textarea className="mt-1 h-24 w-full rounded-lg border border-clay px-2 py-1.5" value={form.steps} onChange={(e) => setForm({ ...form, steps: e.target.value })} />
          </label>
          <div>
            <p className="text-sm">食材 *</p>
            <FieldError text={errs.items} />
            {form.items.map((it, i) => (
              <div key={i} className="mt-2 grid grid-cols-12 items-start gap-2">
                <select
                  className="col-span-5 rounded-lg border border-clay px-2 py-1.5 text-sm"
                  value={it.ingredient_id}
                  onChange={(e) => {
                    const next = [...form.items];
                    const id = Number(e.target.value);
                    const ing = ings.find((x) => x.id === id);
                    next[i] = { ...it, ingredient_id: id, unit: ing?.default_unit || it.unit };
                    setForm({ ...form, items: next });
                  }}
                >
                  <option value={0}>选择食材</option>
                  {ings.map((ing) => <option key={ing.id} value={ing.id}>{ing.name}</option>)}
                </select>
                <input className="col-span-2 rounded-lg border border-clay px-2 py-1.5 text-sm" value={it.quantity} onChange={(e) => {
                  const next = [...form.items]; next[i] = { ...it, quantity: e.target.value }; setForm({ ...form, items: next });
                }} />
                <input className="col-span-3 rounded-lg border border-clay px-2 py-1.5 text-sm" value={it.unit} onChange={(e) => {
                  const next = [...form.items]; next[i] = { ...it, unit: e.target.value }; setForm({ ...form, items: next });
                }} />
                <button type="button" className="col-span-2 text-xs text-chili" onClick={() => setForm({ ...form, items: form.items.filter((_, j) => j !== i) })}>移除</button>
                <FieldError text={errs[`items.${i}.ingredient_id`] || errs[`items.${i}.unit`] || errs[`items.${i}.quantity`]} />
              </div>
            ))}
            <button type="button" className="mt-2 text-sm text-leaf" onClick={() => setForm({ ...form, items: [...form.items, { ingredient_id: 0, quantity: "1", unit: "个", optional: false }] })}>+ 加一味</button>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setOpen(false)} className="rounded-full px-3 py-1 text-soot">取消</button>
            <button type="submit" className="rounded-full bg-terracotta px-4 py-1.5 text-paper">保存</button>
          </div>
        </form>
      </Dialog>

      <Dialog open={!!del} title="删除这道菜？" onClose={() => setDel(null)}>
        <p className="text-sm text-soot">删除后不可恢复。</p>
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" onClick={() => setDel(null)}>取消</button>
          <button
            type="button"
            className="rounded-full bg-chili px-3 py-1 text-paper"
            onClick={async () => {
              if (!del) return;
              await api.deleteRecipe(del.id);
              setDel(null);
              await load();
            }}
          >
            确认删除
          </button>
        </div>
      </Dialog>
    </div>
  );
}
