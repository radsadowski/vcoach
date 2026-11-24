import { useQuery } from "@tanstack/react-query";
import { getGoalProgress } from "@vcoach/shared";
import { formatNumber } from "../../lib/utils";

export function GoalProgress({ goalId }: { goalId: string }) {
  const progress = useQuery({
    queryKey: ["goal-progress", goalId],
    queryFn: () => getGoalProgress(goalId),
  });

  if (progress.isPending) return <p className="text-sm text-neutral-500">Loading progress…</p>;
  if (progress.isError || !progress.data) return null;

  const data = progress.data;

  if (data.run) {
    const run = data.run;
    return (
      <div className="grid grid-cols-3 gap-4 text-sm">
        <Metric label="Longest run (30d)" value={`${formatNumber(run.longestRunKm)} km`} />
        <Metric
          label="Best pace (30d)"
          value={run.bestPaceSecPerKm ? formatPace(run.bestPaceSecPerKm) : "—"}
        />
        <Metric
          label="This week"
          value={`${formatNumber(run.weeklyVolumes.at(-1)?.distanceKm ?? null)} km`}
        />
      </div>
    );
  }

  if (data.climb) {
    const climb = data.climb;
    return (
      <div className="grid grid-cols-3 gap-4 text-sm">
        <Metric label="Hardest sent (30d)" value={climb.hardestSent || "—"} />
        <Metric
          label="Gap to target"
          value={climb.rankGap === 0 ? "at target!" : `${climb.rankGap} grade${climb.rankGap === 1 ? "" : "s"}`}
        />
        <Metric label="Sessions (30d)" value={`${climb.sessions30d} · ${climb.attempts30d} attempts`} />
      </div>
    );
  }

  return null;
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-neutral-500">{label}</p>
      <p className="mt-0.5 font-semibold text-neutral-200">{value}</p>
    </div>
  );
}

function formatPace(secondsPerKm: number): string {
  const minutes = Math.floor(secondsPerKm / 60);
  const seconds = Math.round(secondsPerKm % 60);
  return `${minutes}:${String(seconds).padStart(2, "0")} /km`;
}
