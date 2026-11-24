import { useQuery } from "@tanstack/react-query";
import { getProfile, listGoals, listPlans } from "@vcoach/shared";
import { Badge } from "../../components/ui/badge";
import { Card, CardContent, CardDescription, CardTitle } from "../../components/ui/card";
import { formatNumber } from "../../lib/utils";

export function DashboardView() {
  const profile = useQuery({ queryKey: ["profile"], queryFn: getProfile });
  const goals = useQuery({ queryKey: ["goals"], queryFn: listGoals });
  const plans = useQuery({ queryKey: ["plans"], queryFn: listPlans });

  const activePlan = plans.data?.find((plan) => plan.status === "active");
  const primaryGoal = goals.data?.find((goal) => goal.isPrimary && goal.status === "active");
  const derived = profile.data?.derived;

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-8">
      <header>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="mt-1 text-sm text-neutral-400">
          {primaryGoal
            ? `Training towards ${primaryGoal.activityType === "run" ? `${primaryGoal.targetValue} km run` : primaryGoal.targetValue}`
            : "No primary goal yet — tell the coach what you're aiming for."}
        </p>
      </header>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardTitle className="text-sm font-medium text-neutral-400">Latest VO₂max</CardTitle>
          <CardContent>
            <span className="text-3xl font-bold">{formatNumber(derived?.vo2max ?? null)}</span>
          </CardContent>
        </Card>
        <Card>
          <CardTitle className="text-sm font-medium text-neutral-400">Longest run (30d)</CardTitle>
          <CardContent>
            <span className="text-3xl font-bold">{formatNumber(derived?.longestRunKm ?? null)}</span>
            <span className="ml-1 text-sm text-neutral-500">km</span>
          </CardContent>
        </Card>
        <Card>
          <CardTitle className="text-sm font-medium text-neutral-400">Weight</CardTitle>
          <CardContent>
            <span className="text-3xl font-bold">{formatNumber(derived?.weight?.latestKg ?? null)}</span>
            <span className="ml-1 text-sm text-neutral-500">kg</span>
            {derived?.weight && (
              <p className="mt-1 text-xs text-neutral-500">
                {derived.weight.changeKg >= 0 ? "+" : ""}
                {formatNumber(derived.weight.changeKg)} kg vs 30d ago
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardTitle>Weekly run volume</CardTitle>
        <CardDescription>Last four weeks</CardDescription>
        <CardContent>
          {derived?.weeklyVolumes?.length ? (
            <div className="flex h-32 items-end gap-3">
              {derived.weeklyVolumes.map((week) => {
                const max = Math.max(...derived.weeklyVolumes.map((w) => w.distanceKm), 1);
                return (
                  <div key={week.week} className="flex w-14 flex-col items-center gap-1">
                    <div
                      className="w-full rounded-t bg-emerald-700/70"
                      style={{ height: `${Math.max((week.distanceKm / max) * 100, 4)}%` }}
                    />
                    <span className="text-xs text-neutral-500">{formatNumber(week.distanceKm, 0)}</span>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-neutral-500">No runs synced yet.</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardTitle>Active goals</CardTitle>
        <CardContent className="space-y-2">
          {goals.data?.filter((goal) => goal.status === "active").length ? (
            goals.data
              ?.filter((goal) => goal.status === "active")
              .map((goal) => (
                <div key={goal.id} className="flex items-center justify-between text-sm">
                  <span>
                    {goal.activityType === "run" && `${goal.targetValue} km run`}
                    {goal.activityType === "boulder" && `${goal.targetValue} boulder`}
                    {goal.activityType === "lead" && `${goal.targetValue} lead route`}
                  </span>
                  {goal.isPrimary && <Badge variant="info">primary</Badge>}
                </div>
              ))
          ) : (
            <p className="text-sm text-neutral-500">No active goals.</p>
          )}
        </CardContent>
      </Card>

      {activePlan && (
        <Card>
          <CardTitle>Current plan</CardTitle>
          <CardDescription>{activePlan.title}</CardDescription>
          <CardContent>
            <Badge variant="success">v{activePlan.version} · active</Badge>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
