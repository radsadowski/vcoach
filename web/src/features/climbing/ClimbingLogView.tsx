import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createClimbingSession, deleteClimbingSession, listClimbingSessions } from "@vcoach/shared";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { formatDate } from "../../lib/utils";

export function ClimbingLogView() {
  const queryClient = useQueryClient();
  const sessions = useQuery({ queryKey: ["climbing-sessions"], queryFn: () => listClimbingSessions() });

  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [venue, setVenue] = useState("");
  const [discipline, setDiscipline] = useState("boulder");
  const [hardestSent, setHardestSent] = useState("");
  const [hardestAttempted, setHardestAttempted] = useState("");
  const [attempts, setAttempts] = useState("0");
  const [rpe, setRpe] = useState("");
  const [error, setError] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: createClimbingSession,
    onSuccess: () => {
      setError(null);
      setVenue("");
      setHardestSent("");
      setHardestAttempted("");
      setAttempts("0");
      setRpe("");
      queryClient.invalidateQueries({ queryKey: ["climbing-sessions"] });
    },
    onError: (err) => setError(err instanceof Error ? err.message : "failed to log session"),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteClimbingSession(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["climbing-sessions"] }),
  });

  const submit = () => {
    create.mutate({
      sessionDate: date,
      venue: venue || undefined,
      discipline: discipline as "boulder" | "lead",
      hardestSent,
      hardestAttempted: hardestAttempted || hardestSent,
      attempts: attempts ? Number(attempts) : undefined,
      rpe: rpe ? Number(rpe) : undefined,
    });
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-8">
      <header>
        <h1 className="text-2xl font-bold tracking-tight">Climbing log</h1>
        <p className="mt-1 text-sm text-neutral-400">
          Manual sessions — the source for boulder and lead goal progress.
        </p>
      </header>

      <Card>
        <CardTitle>Log a session</CardTitle>
        <CardContent className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div className="space-y-1">
            <Label htmlFor="date">Date</Label>
            <Input id="date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="venue">Venue</Label>
            <Input id="venue" value={venue} onChange={(e) => setVenue(e.target.value)} placeholder="gym / board" />
          </div>
          <div className="space-y-1">
            <Label htmlFor="discipline">Discipline</Label>
            <select
              id="discipline"
              value={discipline}
              onChange={(e) => setDiscipline(e.target.value)}
              className="h-9 w-full rounded-md border border-neutral-700 bg-neutral-900 px-3 text-sm text-neutral-100"
            >
              <option value="boulder">Boulder</option>
              <option value="lead">Lead</option>
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="sent">Hardest sent</Label>
            <Input id="sent" value={hardestSent} onChange={(e) => setHardestSent(e.target.value)} placeholder="6B" />
          </div>
          <div className="space-y-1">
            <Label htmlFor="attempted">Hardest attempted</Label>
            <Input id="attempted" value={hardestAttempted} onChange={(e) => setHardestAttempted(e.target.value)} placeholder="6B+" />
          </div>
          <div className="space-y-1">
            <Label htmlFor="attempts">Attempts</Label>
            <Input id="attempts" type="number" min={0} value={attempts} onChange={(e) => setAttempts(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="rpe">RPE (1–10)</Label>
            <Input id="rpe" type="number" min={1} max={10} value={rpe} onChange={(e) => setRpe(e.target.value)} />
          </div>
          <div className="flex items-end">
            <Button onClick={submit} disabled={create.isPending || !hardestSent}>
              {create.isPending ? "Saving…" : "Log session"}
            </Button>
          </div>
        </CardContent>
        {error && <CardContent><p className="text-sm text-red-400">{error}</p></CardContent>}
      </Card>

      <Card>
        <CardTitle>Sessions</CardTitle>
        <CardContent className="space-y-2">
          {sessions.data?.length ? (
            sessions.data.map((session) => (
              <div
                key={session.id}
                className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/50 px-3 py-2"
              >
                <div>
                  <p className="text-sm text-neutral-200">
                    <span className="font-medium">{formatDate(session.sessionDate)}</span>
                    {session.venue ? ` · ${session.venue}` : ""} · sent{" "}
                    <span className="font-semibold text-emerald-400">{session.hardestSent}</span>
                    {session.hardestAttempted !== session.hardestSent && ` (tried ${session.hardestAttempted})`}
                  </p>
                  <p className="text-xs text-neutral-500">
                    {session.attempts} attempts{session.rpe ? ` · RPE ${session.rpe}` : ""}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={session.discipline === "boulder" ? "info" : "default"}>
                    {session.discipline}
                  </Badge>
                  <Button size="sm" variant="ghost" onClick={() => remove.mutate(session.id)}>
                    Delete
                  </Button>
                </div>
              </div>
            ))
          ) : (
            <p className="text-sm text-neutral-500">No sessions logged yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
