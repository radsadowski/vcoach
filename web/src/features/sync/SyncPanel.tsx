import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getSyncStatus, syncNow } from "@vcoach/shared";
import { Button } from "../../components/ui/button";
import { cn, formatDate } from "../../lib/utils";

export function SyncPanel() {
  const queryClient = useQueryClient();
  const status = useQuery({ queryKey: ["sync-status"], queryFn: getSyncStatus, refetchInterval: 60_000 });

  const sync = useMutation({
    mutationFn: syncNow,
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const hasError = status.data?.sources.some((source) => source.lastError);
  const lastSync = status.data?.sources
    .map((source) => source.lastSyncedAt)
    .filter(Boolean)
    .sort()
    .at(-1);

  return (
    <div className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-neutral-400">Garmin sync</span>
        <Button size="sm" variant="outline" onClick={() => sync.mutate()} disabled={sync.isPending}>
          {sync.isPending ? "Syncing…" : "Sync now"}
        </Button>
      </div>
      <p className="mt-2 text-xs text-neutral-500">
        {hasError ? "Last error: " : "Last sync: "}
        <span className={cn(hasError ? "text-amber-400" : "text-neutral-400")}>
          {hasError
            ? (status.data?.sources.find((source) => source.lastError)?.lastError ?? "unknown")
            : formatDate(lastSync ?? null)}
        </span>
      </p>
      {sync.data && (sync.data.errors?.length ?? 0) > 0 && (
        <p className="mt-1 text-xs text-amber-500">{sync.data.errors?.join("; ")}</p>
      )}
      {sync.data && (sync.data.errors?.length ?? 0) === 0 && (
        <p className="mt-1 text-xs text-emerald-400">
          Synced {sync.data.activitiesSynced} activities, {sync.data.healthDaysSynced} health days.
        </p>
      )}
    </div>
  );
}
