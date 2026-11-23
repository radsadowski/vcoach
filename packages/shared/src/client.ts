import type {
  Athlete,
  AthleteWithDerived,
  ChatMessage,
  ChatSession,
  ClimbingSession,
  CreateClimbingSessionInput,
  CreateGoalInput,
  CreatePlanInput,
  Goal,
  HealthStatus,
  Plan,
  PlanWithSessions,
  SyncResult,
  SyncStatus,
  UpdateAthleteInput,
  UpdateClimbingSessionInput,
  UpdateGoalInput,
} from "./types";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: string,
  ) {
    super(`api error ${status}: ${body}`);
    this.name = "ApiError";
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "content-type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    throw new ApiError(res.status, await res.text());
  }
  return (await res.json()) as T;
}

export function getHealth(): Promise<HealthStatus> {
  return apiFetch<HealthStatus>("/healthz");
}

export function getProfile(): Promise<AthleteWithDerived> {
  return apiFetch<AthleteWithDerived>("/api/profile");
}

export function updateProfile(input: UpdateAthleteInput): Promise<Athlete> {
  return apiFetch<Athlete>("/api/profile", { method: "PUT", body: JSON.stringify(input) });
}

export function listGoals(): Promise<Goal[]> {
  return apiFetch<Goal[]>("/api/goals");
}

export function createGoal(input: CreateGoalInput): Promise<Goal> {
  return apiFetch<Goal>("/api/goals", { method: "POST", body: JSON.stringify(input) });
}

export function updateGoal(id: string, input: UpdateGoalInput): Promise<Goal> {
  return apiFetch<Goal>(`/api/goals/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export async function deleteGoal(id: string): Promise<void> {
  const res = await fetch(`/api/goals/${id}`, { method: "DELETE" });
  if (!res.ok) {
    throw new ApiError(res.status, await res.text());
  }
}

export function listPlans(): Promise<Plan[]> {
  return apiFetch<Plan[]>("/api/plans");
}

export function getPlan(id: string): Promise<PlanWithSessions> {
  return apiFetch<PlanWithSessions>(`/api/plans/${id}`);
}

export function createPlan(input: CreatePlanInput): Promise<PlanWithSessions> {
  return apiFetch<PlanWithSessions>("/api/plans", { method: "POST", body: JSON.stringify(input) });
}

export async function archivePlan(id: string): Promise<void> {
  const res = await fetch(`/api/plans/${id}`, { method: "DELETE" });
  if (!res.ok) {
    throw new ApiError(res.status, await res.text());
  }
}

export function listClimbingSessions(from?: string, to?: string): Promise<ClimbingSession[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const query = params.size > 0 ? `?${params.toString()}` : "";
  return apiFetch<ClimbingSession[]>(`/api/climbing-sessions${query}`);
}

export function createClimbingSession(input: CreateClimbingSessionInput): Promise<ClimbingSession> {
  return apiFetch<ClimbingSession>("/api/climbing-sessions", { method: "POST", body: JSON.stringify(input) });
}

export function updateClimbingSession(id: string, input: UpdateClimbingSessionInput): Promise<ClimbingSession> {
  return apiFetch<ClimbingSession>(`/api/climbing-sessions/${id}`, { method: "PUT", body: JSON.stringify(input) });
}

export async function deleteClimbingSession(id: string): Promise<void> {
  const res = await fetch(`/api/climbing-sessions/${id}`, { method: "DELETE" });
  if (!res.ok) {
    throw new ApiError(res.status, await res.text());
  }
}

export function listChatSessions(): Promise<ChatSession[]> {
  return apiFetch<ChatSession[]>("/api/chat-sessions");
}

export function createChatSession(title?: string): Promise<ChatSession> {
  return apiFetch<ChatSession>("/api/chat-sessions", { method: "POST", body: JSON.stringify({ title }) });
}

export function listChatMessages(sessionId: string): Promise<ChatMessage[]> {
  return apiFetch<ChatMessage[]>(`/api/chat-sessions/${sessionId}/messages`);
}

export function syncNow(): Promise<SyncResult> {
  return apiFetch<SyncResult>("/api/sync", { method: "POST" });
}

export function getSyncStatus(): Promise<SyncStatus> {
  return apiFetch<SyncStatus>("/api/sync/status");
}
