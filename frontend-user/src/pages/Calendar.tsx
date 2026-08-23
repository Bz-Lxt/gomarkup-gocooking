import { useCallback, useEffect, useMemo, useState } from "react";
import { DndContext, DragOverlay, PointerSensor, useDraggable, useDroppable, useSensor, useSensors, type DragEndEvent, type DragStartEvent } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import { Link } from "react-router-dom";
import { api } from "../api";
import { SLOT_META, addDays, fmt, parseYMD, startOfWeek, todayYMD, weekDays } from "../dates";
import type { Recipe, Slot, WeekPlan } from "../types";
import { Dialog, ToastHost, useToast } from "../ui";

type DragData =
  | { kind: "recipe"; recipe: Recipe }
  | { kind: "slot"; slot: Slot };

export default function Calendar() {
  const toast = useToast();
  const [week, setWeek] = useState(() => fmt(startOfWeek(new Date())));
  const [plan, setPlan] = useState<WeekPlan | null>(null);
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [q, setQ] = useState("");
  const [active, setActive] = useState<DragData | null>(null);
  const [confirmClear, setConfirmClear] = useState(false);
  const days = useMemo(() => weekDays(week), [week]);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  const load = useCallback(async () => {
    try {
      const [p, r] = await Promise.all([api.week(week), api.recipes(q)]);
      setPlan(p);
      setRecipes(r);
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "加载失败");
    }
  }, [week, q]);

  useEffect(() => {
    void load();
  }, [load]);

  async function onDragEnd(e: DragEndEvent) {
    const data = e.active.data.current as DragData | undefined;
    const overId = String(e.over?.id || "");
    setActive(null);
    if (!data) return;
    if (overId === "trash") {
      if (data.kind === "slot") {
        try {
          await api.deleteSlot(data.slot.id);
          await load();
        } catch (err) {
          toast.show(err instanceof Error ? err.message : "删除失败");
        }
      }
      return;
    }
    if (!overId.startsWith("cell:")) {
      if (data.kind === "slot") {
        try {
          await api.deleteSlot(data.slot.id);
          await load();
        } catch (err) {
          toast.show(err instanceof Error ? err.message : "移出失败");
        }
      }
      return;
    }
    const [, date, slot] = overId.split(":");
    try {
      if (data.kind === "recipe") {
        await api.addSlot({ date, slot, recipe_id: data.recipe.id, servings_multiplier: 1 });
      } else {
        await api.patchSlot(data.slot.id, { date, slot });
      }
      await load();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "排餐失败");
    }
  }

  return (
    <DndContext
      sensors={sensors}
      onDragStart={(e: DragStartEvent) => setActive((e.active.data.current as DragData) || null)}
      onDragEnd={onDragEnd}
      onDragCancel={() => setActive(null)}
    >
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <h1 className="font-display text-2xl">本周吃什么</h1>
        <div className="flex items-center gap-2 text-sm">
          <button type="button" className="rounded-full bg-clay px-3 py-1" onClick={() => setWeek(fmt(addDays(parseYMD(week), -7)))}>上一周</button>
          <button type="button" className="rounded-full bg-ink px-3 py-1 text-paper" onClick={() => setWeek(fmt(startOfWeek(new Date())))}>回到本周</button>
          <button type="button" className="rounded-full bg-clay px-3 py-1" onClick={() => setWeek(fmt(addDays(parseYMD(week), 7)))}>下一周</button>
        </div>
        <span className="text-sm text-soot">{plan?.week_start} → {plan?.week_end}</span>
        <div className="ml-auto flex gap-2 text-sm">
          <button type="button" className="rounded-full border border-clay px-3 py-1" onClick={() => setConfirmClear(true)}>清空本周</button>
          <button
            type="button"
            className="rounded-full border border-clay px-3 py-1"
            onClick={async () => {
              try {
                await api.copyNext(week);
                toast.show("已复制到下一周");
              } catch (e) {
                toast.show(e instanceof Error ? e.message : "复制失败");
              }
            }}
          >
            复制到下周
          </button>
          <Link to="/shopping" className="rounded-full bg-terracotta px-3 py-1 text-paper">生成买菜清单</Link>
        </div>
      </div>

      <div className="flex flex-col gap-4 lg:flex-row">
        <aside className="w-full shrink-0 lg:w-72">
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="搜索菜谱…"
            className="mb-3 w-full rounded-xl border border-clay bg-white/40 px-3 py-2 text-sm"
          />
          <div className="flex max-h-[70vh] flex-col gap-2 overflow-auto pr-1">
            {recipes.map((r) => (
              <RecipeCard key={r.id} recipe={r} />
            ))}
          </div>
        </aside>

        <section className="min-w-0 flex-1">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-7">
            {days.map((d) => (
              <DayCol key={d.date} day={d} slots={(plan?.slots || []).filter((s) => s.date === d.date)} onChanged={load} />
            ))}
          </div>
          <TrashZone />
        </section>
      </div>

      <DragOverlay>
        {active?.kind === "recipe" ? <CardFace name={active.recipe.name} tag={active.recipe.cuisine_tag} lift /> : null}
        {active?.kind === "slot" ? <CardFace name={active.slot.recipe.name} tag={`${active.slot.servings_multiplier}x`} lift /> : null}
      </DragOverlay>

      <Dialog open={confirmClear} title="清空本周排期？" onClose={() => setConfirmClear(false)}>
        <p className="text-sm text-soot">这会删除本周全部餐位，不可撤销。</p>
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" className="rounded-full px-3 py-1 text-soot" onClick={() => setConfirmClear(false)}>取消</button>
          <button
            type="button"
            className="rounded-full bg-chili px-3 py-1 text-paper"
            onClick={async () => {
              try {
                await api.clearWeek(week);
                setConfirmClear(false);
                await load();
              } catch (e) {
                toast.show(e instanceof Error ? e.message : "清空失败");
              }
            }}
          >
            确认清空
          </button>
        </div>
      </Dialog>
    </DndContext>
  );
}

function RecipeCard({ recipe }: { recipe: Recipe }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `recipe:${recipe.id}`,
    data: { kind: "recipe", recipe } satisfies DragData,
  });
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Translate.toString(transform), opacity: isDragging ? 0.35 : 1 }}
      {...listeners}
      {...attributes}
    >
      <CardFace name={recipe.name} tag={recipe.cuisine_tag || "家常"} />
    </div>
  );
}

function DayCol({ day, slots, onChanged }: { day: { date: string; label: string; md: string }; slots: Slot[]; onChanged: () => void }) {
  const isToday = day.date === todayYMD();
  return (
    <div className={`rounded-2xl border p-2 ${isToday ? "border-terracotta bg-terracotta/5" : "border-clay/80 bg-white/30"}`}>
      <div className="mb-2 flex items-baseline justify-between">
        <span className="font-display">{day.label}</span>
        <span className="text-xs text-soot">{day.md}</span>
      </div>
      {SLOT_META.map((s) => (
        <SlotCell key={s.key} date={day.date} slot={s.key} label={s.label} items={slots.filter((x) => x.slot === s.key)} onChanged={onChanged} />
      ))}
    </div>
  );
}

function SlotCell({ date, slot, label, items, onChanged }: { date: string; slot: string; label: string; items: Slot[]; onChanged: () => void }) {
  const { setNodeRef, isOver } = useDroppable({ id: `cell:${date}:${slot}` });
  return (
    <div
      ref={setNodeRef}
      className={`mb-2 min-h-[88px] rounded-xl border border-dashed p-1.5 ${isOver ? "border-leaf bg-leaf/10" : "border-clay"}`}
    >
      <p className="mb-1 text-[11px] text-soot">{label}</p>
      <div className="flex flex-col gap-1">
        {items.map((it) => (
          <PlacedCard key={it.id} slot={it} onChanged={onChanged} />
        ))}
      </div>
    </div>
  );
}

function PlacedCard({ slot, onChanged }: { slot: Slot; onChanged: () => void }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `slot:${slot.id}`,
    data: { kind: "slot", slot } satisfies DragData,
  });
  return (
    <div ref={setNodeRef} style={{ transform: CSS.Translate.toString(transform), opacity: isDragging ? 0.3 : 1 }} {...listeners} {...attributes}>
      <div className="rounded-lg bg-paper px-2 py-1 shadow-sm">
        <p className="truncate text-xs font-medium">{slot.recipe?.name}</p>
        <label className="mt-1 flex items-center gap-1 text-[10px] text-soot" onPointerDown={(e) => e.stopPropagation()}>
          份数
          <select
            value={slot.servings_multiplier}
            onChange={async (e) => {
              await api.patchSlot(slot.id, { servings_multiplier: Number(e.target.value) });
              onChanged();
            }}
            className="rounded border border-clay bg-white/70 px-1 py-0.5"
          >
            {[0.5, 1, 1.5, 2, 3, 4].map((n) => (
              <option key={n} value={n}>{n}x</option>
            ))}
          </select>
        </label>
      </div>
    </div>
  );
}

function TrashZone() {
  const { setNodeRef, isOver } = useDroppable({ id: "trash" });
  return (
    <div
      ref={setNodeRef}
      className={`mt-3 rounded-xl border border-dashed px-4 py-3 text-center text-sm ${isOver ? "border-chili bg-chili/10 text-chili" : "border-clay text-soot"}`}
    >
      把已排菜谱拖到这里，或拖出日历，即可移除
    </div>
  );
}

function CardFace({ name, tag, lift }: { name: string; tag: string; lift?: boolean }) {
  return (
    <div className={`rounded-xl border-l-[6px] border-terracotta bg-paper px-3 py-2 shadow-ticket ${lift ? "rotate-2 shadow-lift" : ""}`}>
      <p className="font-medium leading-snug">{name}</p>
      <p className="text-[11px] text-soot">{tag}</p>
    </div>
  );
}
