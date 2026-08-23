import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { clearToken } from "../api";

const links = [
  { to: "/", label: "本周排餐" },
  { to: "/shopping", label: "买菜清单" },
  { to: "/recipes", label: "私房菜谱" },
  { to: "/pantry", label: "冰箱库存" },
  { to: "/settings", label: "常备设置" },
];

export default function Layout() {
  const nav = useNavigate();
  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-40 border-b border-clay/80 bg-paper/90 backdrop-blur">
        <div className="flex w-full items-center gap-6 px-4 py-3 md:px-6">
          <div className="font-display text-xl tracking-wide">
            <span className="text-terracotta">灶</span>下清单
          </div>
          <nav className="hidden flex-1 items-center gap-1 md:flex">
            {links.map((l) => (
              <NavLink
                key={l.to}
                to={l.to}
                end={l.to === "/"}
                className={({ isActive }) =>
                  `rounded-full px-3 py-1.5 text-sm ${isActive ? "bg-ink text-paper" : "text-soot hover:bg-clay/70"}`
                }
              >
                {l.label}
              </NavLink>
            ))}
          </nav>
          <button
            type="button"
            className="ml-auto text-sm text-soot hover:text-chili"
            onClick={() => {
              clearToken();
              nav("/login");
            }}
          >
            退出
          </button>
        </div>
        <nav className="flex gap-2 overflow-x-auto px-4 pb-3 md:hidden">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.to === "/"}
              className={({ isActive }) =>
                `whitespace-nowrap rounded-full px-3 py-1 text-sm ${isActive ? "bg-ink text-paper" : "bg-clay/60 text-soot"}`
              }
            >
              {l.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className="w-full px-3 py-4 md:px-6 md:py-6">
        <Outlet />
      </main>
    </div>
  );
}
