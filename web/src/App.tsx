import { NavLink, Outlet } from "react-router-dom";
import { SyncPanel } from "./features/sync/SyncPanel";
import { cn } from "./lib/utils";

const nav = [
  { to: "/", label: "Dashboard", end: true },
  { to: "/chat", label: "Chat" },
  { to: "/plan", label: "Plan" },
  { to: "/goals", label: "Goals" },
  { to: "/climbing", label: "Climbing log" },
  { to: "/profile", label: "Profile" },
];

export default function App() {
  return (
    <div className="flex min-h-screen bg-neutral-950 text-neutral-100">
      <aside className="flex w-56 shrink-0 flex-col border-r border-neutral-800 bg-neutral-900/40">
        <div className="px-5 py-6">
          <span className="text-xl font-bold tracking-tight">vcoach</span>
          <p className="mt-1 text-xs text-neutral-500">personal AI training coach</p>
        </div>
        <nav className="flex flex-col gap-1 px-3">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-neutral-800 text-neutral-100"
                    : "text-neutral-400 hover:bg-neutral-800/60 hover:text-neutral-200",
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="mt-auto p-3">
          <SyncPanel />
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  );
}
