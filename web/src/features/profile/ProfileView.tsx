import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getProfile, updateProfile } from "@vcoach/shared";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { formatNumber } from "../../lib/utils";

export function ProfileView() {
  const queryClient = useQueryClient();
  const profile = useQuery({ queryKey: ["profile"], queryFn: getProfile });

  const [birthYear, setBirthYear] = useState("");
  const [sex, setSex] = useState("");
  const [weightKg, setWeightKg] = useState("");
  const [runningExperience, setRunningExperience] = useState("");
  const [climbingExperience, setClimbingExperience] = useState("");
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile.data) {
      setBirthYear(profile.data.birthYear?.toString() ?? "");
      setSex(profile.data.sex ?? "");
      setWeightKg(profile.data.weightKg?.toString() ?? "");
      setRunningExperience(profile.data.runningExperience ?? "");
      setClimbingExperience(profile.data.climbingExperience ?? "");
    }
  }, [profile.data]);

  const save = useMutation({
    mutationFn: updateProfile,
    onSuccess: () => {
      setSaved(true);
      setError(null);
      queryClient.invalidateQueries({ queryKey: ["profile"] });
    },
    onError: (err) => {
      setSaved(false);
      setError(err instanceof Error ? err.message : "failed to save");
    },
  });

  const submit = () => {
    save.mutate({
      birthYear: birthYear ? Number(birthYear) : undefined,
      sex: (sex || undefined) as never,
      weightKg: weightKg ? Number(weightKg) : undefined,
      runningExperience: (runningExperience || undefined) as never,
      climbingExperience: (climbingExperience || undefined) as never,
    });
  };

  const derived = profile.data?.derived;

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-8">
      <header>
        <h1 className="text-2xl font-bold tracking-tight">Profile</h1>
        <p className="mt-1 text-sm text-neutral-400">
          Manual basics; everything else is derived from your Garmin data.
        </p>
      </header>

      <Card>
        <CardTitle>Basics</CardTitle>
        <CardDescription>The coach uses these to calibrate training advice.</CardDescription>
        <CardContent className="grid grid-cols-2 gap-3 md:grid-cols-3">
          <div className="space-y-1">
            <Label htmlFor="birthYear">Birth year</Label>
            <Input id="birthYear" type="number" value={birthYear} onChange={(e) => setBirthYear(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="sex">Sex</Label>
            <select
              id="sex"
              value={sex}
              onChange={(e) => setSex(e.target.value)}
              className="h-9 w-full rounded-md border border-neutral-700 bg-neutral-900 px-3 text-sm text-neutral-100"
            >
              <option value="">—</option>
              <option value="male">male</option>
              <option value="female">female</option>
              <option value="other">other</option>
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="weight">Weight (kg)</Label>
            <Input id="weight" type="number" step="0.1" value={weightKg} onChange={(e) => setWeightKg(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="running">Running experience</Label>
            <select
              id="running"
              value={runningExperience}
              onChange={(e) => setRunningExperience(e.target.value)}
              className="h-9 w-full rounded-md border border-neutral-700 bg-neutral-900 px-3 text-sm text-neutral-100"
            >
              <option value="">—</option>
              <option value="beginner">beginner</option>
              <option value="intermediate">intermediate</option>
              <option value="advanced">advanced</option>
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="climbing">Climbing experience</Label>
            <select
              id="climbing"
              value={climbingExperience}
              onChange={(e) => setClimbingExperience(e.target.value)}
              className="h-9 w-full rounded-md border border-neutral-700 bg-neutral-900 px-3 text-sm text-neutral-100"
            >
              <option value="">—</option>
              <option value="beginner">beginner</option>
              <option value="intermediate">intermediate</option>
              <option value="advanced">advanced</option>
            </select>
          </div>
          <div className="flex items-end gap-2">
            <Button onClick={submit} disabled={save.isPending}>
              {save.isPending ? "Saving…" : "Save"}
            </Button>
            {saved && <span className="text-xs text-emerald-400">Saved</span>}
          </div>
        </CardContent>
        {error && <CardContent><p className="text-sm text-red-400">{error}</p></CardContent>}
      </Card>

      <Card>
        <CardTitle>From Garmin</CardTitle>
        <CardDescription>Derived from synced activities and daily health.</CardDescription>
        <CardContent className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
          <Derived label="Max HR" value={derived?.heartRate ? `${derived.heartRate.maxHeartRate} bpm` : null} />
          <Derived label="Resting HR" value={derived?.heartRate ? `${derived.heartRate.restingHr} bpm` : null} />
          <Derived label="VO₂max" value={derived?.vo2max ? formatNumber(derived.vo2max) : null} />
          <Derived label="Longest run (30d)" value={derived?.longestRunKm ? `${formatNumber(derived.longestRunKm)} km` : null} />
        </CardContent>
      </Card>
    </div>
  );
}

function Derived({ label, value }: { label: string; value: string | null }) {
  return (
    <div>
      <p className="text-xs text-neutral-500">{label}</p>
      <p className="mt-0.5 font-semibold text-neutral-200">{value ?? "—"}</p>
    </div>
  );
}
