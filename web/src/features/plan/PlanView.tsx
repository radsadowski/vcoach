import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { archivePlan, getPlan, listPlans } from "@vcoach/shared";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "../../components/ui/card";
import { formatDate } from "../../lib/utils";

const statusVariant = {
  planned: "default",
  completed: "success",
  missed: "danger",
  skipped: "warning",
} as const;

export function PlanView() {
  const queryClient = useQueryClient();
  const plans = useQuery({ queryKey: ["plans"], queryFn: listPlans });
  const activePlan = plans.data?.find((plan) => plan.status === "active");
  const plan = useQuery({
    queryKey: ["plan", activePlan?.id],
    queryFn: () => getPlan(activePlan!.id),
    enabled: !!activePlan,
  });

  const archive = useMutation({
    mutationFn: (planId: string) => archivePlan(planId),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-8">
      <header className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Training plan</h1>
          <p className="mt-1 text-sm text-neutral-400">
            Created and adapted by the coach through chat — chat is the only editor.
          </p>
        </div>
        {activePlan && (
          <Button variant="destructive" size="sm" onClick={() => archive.mutate(activePlan.id)}>
            Archive plan
          </Button>
        )}
      </header>

      {!activePlan && (
        <Card>
          <CardContent>
            <p className="text-sm text-neutral-400">
              No active plan. Ask the coach in chat —{" "}
              <span className="text-neutral-300">"Build me a 12-week plan for my goal."</span>
            </p>
          </CardContent>
        </Card>
      )}

      {plan.data && (
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>{plan.data.title}</CardTitle>
              <CardDescription>
                v{plan.data.version} · since {formatDate(plan.data.createdAt)}
                {plan.data.notes ? ` · ${plan.data.notes}` : ""}
              </CardDescription>
            </div>
            <Badge variant="success">active</Badge>
          </div>
          <CardContent className="space-y-5">
            {plan.data.weeks.map((week) => (
              <div key={week.id}>
                <div className="flex items-baseline justify-between">
                  <h4 className="text-sm font-semibold text-neutral-200">
                    Week {week.weekNumber}
                    {week.focus ? ` — ${week.focus}` : ""}
                  </h4>
                  <span className="text-xs text-neutral-500">{formatDate(week.startDate)}</span>
                </div>
                <div className="mt-2 space-y-2">
                  {week.sessions.length === 0 && (
                    <p className="text-xs text-neutral-600">Rest week.</p>
                  )}
                  {week.sessions.map((session) => (
                    <div
                      key={session.id}
                      className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/50 px-3 py-2"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm text-neutral-200">
                          <span className="font-medium capitalize">{session.sport}</span>{" "}
                          · {session.sessionType}
                          {session.targetDistanceKm ? ` · ${session.targetDistanceKm} km` : ""}
                          {session.targetDurationMin ? ` · ${session.targetDurationMin} min` : ""}
                        </p>
                        {session.description && (
                          <p className="truncate text-xs text-neutral-500">{session.description}</p>
                        )}
                      </div>
                      <div className="ml-3 shrink-0 text-right">
                        <Badge variant={statusVariant[session.status] ?? "default"}>
                          {session.status}
                        </Badge>
                        <p className="mt-1 text-xs text-neutral-500">
                          {formatDate(session.plannedDate)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {(plans.data?.length ?? 0) > 1 && (
        <Card>
          <CardTitle>History</CardTitle>
          <CardContent className="space-y-2">
            {plans.data
              ?.filter((version) => version.status === "archived")
              .map((version) => (
                <div key={version.id} className="flex items-center justify-between text-sm">
                  <span className="text-neutral-300">
                    v{version.version} · {version.title}
                  </span>
                  <span className="text-xs text-neutral-500">{formatDate(version.createdAt)}</span>
                </div>
              ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
