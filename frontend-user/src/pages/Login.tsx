import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, setToken } from "../api";
import { FieldError, ToastHost, useToast } from "../ui";

export default function Login() {
  const nav = useNavigate();
  const toast = useToast();
  const [username, setUsername] = useState("demo");
  const [password, setPassword] = useState("demo123");
  const [errs, setErrs] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    const next: Record<string, string> = {};
    if (!username.trim()) next.username = "请填写用户名";
    if (!password) next.password = "请填写密码";
    setErrs(next);
    if (Object.keys(next).length) {
      toast.show("请先补全登录信息");
      return;
    }
    setBusy(true);
    try {
      const data = await api.login(username.trim(), password);
      setToken(data.token);
      nav("/");
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "登录失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4 paper-grain">
      <ToastHost msg={toast.msg} onClose={toast.clear} />
      <form onSubmit={onSubmit} className="w-full max-w-md rounded-3xl bg-paper p-8 shadow-ticket">
        <p className="text-xs tracking-[0.3em] text-terracotta">GOCOOKING</p>
        <h1 className="mt-2 font-display text-3xl">灶下清单</h1>
        <p className="mt-2 text-sm text-soot">把「今天吃什么」拖进日历，出门只带一张单。</p>
        <label className="mt-8 block text-sm">
          用户名
          <input
            className="mt-1 w-full rounded-xl border border-clay bg-white/50 px-3 py-2 outline-none focus:border-terracotta"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <FieldError text={errs.username} />
        </label>
        <label className="mt-4 block text-sm">
          密码
          <input
            type="password"
            className="mt-1 w-full rounded-xl border border-clay bg-white/50 px-3 py-2 outline-none focus:border-terracotta"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <FieldError text={errs.password} />
        </label>
        <button
          type="submit"
          disabled={busy}
          className="mt-6 w-full rounded-xl bg-terracotta py-2.5 text-paper shadow-lift disabled:opacity-60"
        >
          {busy ? "正在进灶台…" : "进入厨房"}
        </button>
        <p className="mt-3 text-center text-xs text-soot">测试账号 demo / demo123</p>
      </form>
    </div>
  );
}
