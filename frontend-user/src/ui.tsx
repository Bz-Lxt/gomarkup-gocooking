import { useEffect, useState, type ReactNode } from "react";

export function ToastHost({ msg, onClose }: { msg: string; onClose: () => void }) {
  useEffect(() => {
    if (!msg) return;
    const t = setTimeout(onClose, 5000);
    return () => clearTimeout(t);
  }, [msg, onClose]);
  if (!msg) return null;
  return (
    <div className="fixed right-4 top-4 z-[80] flex max-w-sm items-start gap-3 rounded-xl bg-ink px-4 py-3 text-sm text-paper shadow-ticket">
      <p className="flex-1 leading-relaxed">{msg}</p>
      <button type="button" onClick={onClose} className="text-clay hover:text-paper" aria-label="关闭">
        ×
      </button>
    </div>
  );
}

export function Dialog({
  open, title, children, onClose,
}: { open: boolean; title: string; children: ReactNode; onClose: () => void }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-[#2A2118]/40 p-4" onClick={onClose}>
      <div className="w-full max-w-lg rounded-2xl bg-paper p-6 shadow-ticket paper-grain" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-xl">{title}</h3>
          <button type="button" onClick={onClose} className="text-soot hover:text-ink">×</button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function FieldError({ text }: { text?: string }) {
  if (!text) return null;
  return <p className="mt-1 text-xs text-chili">{text}</p>;
}

export function useToast() {
  const [msg, setMsg] = useState("");
  return { msg, show: setMsg, clear: () => setMsg("") };
}
