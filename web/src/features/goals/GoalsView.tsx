import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteGoal, listGoals, updateGoal } from "@vcoach/shared";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "../../components/ui/card";
import { formatDate } from "../../lib/utils";
import { GoalProgress } from "./GoalProgress";

export function GoalsView() {
  const queryClient = useQueryClient();
  const goals = useQuery({ queryKey: ["goals"], queryFn: listGoals });

  const setPrimary = useMutation({
    mutationFn: (goalId: string) => updateGoal(goalId, { isPrimary: true }),
    onSuccess: () => queryClient.invalidateQueries(),
  });
  const remove = useMutation({
    mutationFn: (goalId: string) => deleteGoal(goalId),
    onSuccess: () => queryClient.invalidateQueries(),
  });

  const active = goals.data?.filter((goal) => goal.status === "active") ?? [];
  const past = goals.data?.filter((goal) => goal.status !== "active") ?? [];

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-8">
      <header>
        <h1 className="text-2xl font-bold tracking-tight">Goals</h1>
        <p className="mt-1 text-sm text-neutral-400">
          Created by the coach through chat; one primary goal drives your plan.
        </p>
      </header>

      {active.length === 0 && (
        <Card>
          <CardContent>
            <p className="text-sm text-neutral-400">
              No active goals — tell the coach what you want to achieve.
            </p>
          </CardContent>
        </Card>
      )}

      <div className="space-y-4">
        {active.map((goal) => (
          <Card key={goal.id}>
            <div className="flex items-start justify-between gap-4">
              <div>
                <CardTitle className="flex items-center gap-2">
                  {goalLabel(goal)}
                  {goal.isPrimary && <Badge variant="info">primary</Badge>}
                </CardTitle>
                <CardDescription>
                  {goal.targetDate ? `Target: ${formatDate(goal.targetDate)}` : "No deadline set"}
                  {goal.notes ? ` · ${goal.notes}` : ""}
                </CardDescription>
              </div>
              <div className="flex shrink-0 gap-2">
                {!goal.isPrimary && (
                  <Button size="sm" variant="outline" onClick={() => setPrimary.mutate(goal.id)}>
                    Make primary
                  </Button>
                )}
                <Button size="sm" variant="ghost" onClick={() => remove.mutate(goal.id)}>
                  Delete
                </Button>
              </div>
            </div>
            <CardContent>
              <GoalProgress goalId={goal.id} />
            </CardContent>
          </Card>
        ))}
      </div>

      {past.length > 0 && (
        <Card>
          <CardTitle>History</CardTitle>
          <CardContent className="space-y-2">
            {past.map((goal) => (
              <div key={goal.id} className="flex items-center justify-between text-sm">
                <span className="text-neutral-300">{goalLabel(goal)}</span>
                <Badge variant={goal.status === "achieved" ? "success" : "warning"}>
                  {goal.status}
                </Badge>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function goalLabel(goal: { activityType: string; targetValue: string }): string {
  if (goal.activityType === "run") return `${goal.targetValue} km run`;
  if (goal.activityType === "boulder") return `${goal.targetValue} boulder`;
  return `${goal.targetValue} lead route`;
}
