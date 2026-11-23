import { useEffect, useState } from "react";
import { getHealth, type HealthStatus } from "@vcoach/shared";

export default function App() {
  const [health, setHealth] = useState<HealthStatus | null>(null);

  useEffect(() => {
    let cancelled = false;
    getHealth()
      .then((status) => {
        if (!cancelled) setHealth(status);
      })
      .catch(() => {
        if (!cancelled) setHealth({ status: "unreachable" });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const status = health?.status ?? "checking…";

  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-950 text-neutral-100">
      <div className="text-center">
        <h1 className="text-4xl font-bold tracking-tight">vcoach</h1>
        <p className="mt-2 text-neutral-400">personal AI athletic training coach</p>
        <p className="mt-8 text-sm">
          API:{" "}
          <span className={status === "ok" ? "text-emerald-400" : "text-amber-400"}>{status}</span>
        </p>
      </div>
    </div>
  );
}
