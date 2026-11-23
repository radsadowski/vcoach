export type HealthStatus = {
  status: string;
};

export type Sex = "male" | "female" | "other";
export type ExperienceLevel = "beginner" | "intermediate" | "advanced";

export type Athlete = {
  id: string;
  name: string;
  birthYear: number | null;
  sex: Sex | null;
  weightKg: number | null;
  runningExperience: ExperienceLevel | null;
  climbingExperience: ExperienceLevel | null;
  createdAt: string;
};

export type UpdateAthleteInput = {
  birthYear?: number;
  sex?: Sex;
  weightKg?: number;
  runningExperience?: ExperienceLevel;
  climbingExperience?: ExperienceLevel;
};

export type GoalActivityType = "run" | "boulder" | "lead";
export type GoalStatus = "active" | "achieved" | "abandoned";

export type Goal = {
  id: string;
  athleteId: string;
  activityType: GoalActivityType;
  targetMetric: string;
  targetValue: string;
  targetDate: string | null;
  isPrimary: boolean;
  status: GoalStatus;
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type CreateGoalInput = {
  activityType: GoalActivityType;
  targetMetric: string;
  targetValue: string;
  targetDate?: string;
  notes?: string;
  isPrimary?: boolean;
};

export type UpdateGoalInput = {
  status?: GoalStatus;
  isPrimary?: boolean;
  notes?: string;
};

export type PlanStatus = "active" | "archived";
export type SessionStatus = "planned" | "completed" | "missed" | "skipped";
export type PlanSport = "run" | "boulder" | "lead" | "strength" | "other";

export type Plan = {
  id: string;
  athleteId: string;
  goalId: string | null;
  version: number;
  status: PlanStatus;
  title: string;
  notes: string;
  createdAt: string;
};

export type PlanSession = {
  id: string;
  weekId: string;
  athleteId: string;
  sport: PlanSport;
  sessionType: string;
  description: string;
  plannedDate: string;
  targetDurationMin: number | null;
  targetDistanceKm: number | null;
  targetIntensity: string | null;
  status: SessionStatus;
  matchedActivityId: string | null;
  completedAt: string | null;
};

export type PlanWeek = {
  id: string;
  planId: string;
  weekNumber: number;
  startDate: string;
  focus: string;
  sessions: PlanSession[];
};

export type PlanWithSessions = Plan & {
  weeks: PlanWeek[];
};

export type CreateSessionInput = {
  sport: PlanSport;
  sessionType: string;
  description?: string;
  plannedDate: string;
  targetDurationMin?: number;
  targetDistanceKm?: number;
  targetIntensity?: string;
};

export type CreateWeekInput = {
  weekNumber: number;
  startDate: string;
  focus?: string;
  sessions: CreateSessionInput[];
};

export type CreatePlanInput = {
  goalId?: string;
  title?: string;
  notes?: string;
  weeks: CreateWeekInput[];
};

export type ClimbingDiscipline = "boulder" | "lead";

export type ClimbingSession = {
  id: string;
  athleteId: string;
  sessionDate: string;
  venue: string;
  discipline: ClimbingDiscipline;
  hardestSent: string;
  hardestAttempted: string;
  attempts: number;
  rpe: number | null;
  notes: string;
  createdAt: string;
};

export type CreateClimbingSessionInput = {
  sessionDate: string;
  venue?: string;
  discipline: ClimbingDiscipline;
  hardestSent: string;
  hardestAttempted: string;
  attempts?: number;
  rpe?: number;
  notes?: string;
};

export type UpdateClimbingSessionInput = {
  sessionDate?: string;
  venue?: string;
  discipline?: ClimbingDiscipline;
  hardestSent?: string;
  hardestAttempted?: string;
  attempts?: number;
  rpe?: number;
  notes?: string;
};

export type ChatSession = {
  id: string;
  athleteId: string;
  title: string;
  createdAt: string;
  updatedAt: string;
};

export type ChatRole = "user" | "assistant";

export type ChatMessage = {
  id: string;
  chatSessionId: string;
  athleteId: string;
  role: ChatRole;
  content: string;
  toolCalls: unknown | null;
  createdAt: string;
};

export type HeartRateInfo = {
  maxHeartRate: number;
  restingHr: number;
  zone1Floor: number;
  zone2Floor: number;
  zone3Floor: number;
  zone4Floor: number;
};

export type WeeklyVolume = {
  week: string;
  distanceKm: number;
  sessionCount: number;
};

export type WeightTrend = {
  latestKg: number;
  daysAgoKg: number;
  changeKg: number;
};

export type DerivedProfile = {
  heartRate?: HeartRateInfo;
  vo2max?: number;
  weeklyVolumes: WeeklyVolume[];
  longestRunKm?: number;
  weight?: WeightTrend;
};

export type AthleteWithDerived = Athlete & {
  derived: DerivedProfile;
};

export type SyncResult = {
  activitiesSynced: number;
  healthDaysSynced: number;
  errors?: string[];
};

export type SyncState = {
  source: string;
  athleteId: string;
  lastSyncedAt: string | null;
  cursor: string | null;
  lastError: string | null;
  lastErrorAt: string | null;
};

export type SyncStatus = {
  sources: SyncState[];
};

export type RunProgress = {
  longestRunKm?: number;
  bestPaceSecPerKm?: number;
  weeklyVolumes: { week: string; distanceKm: number; sessionCount: number }[];
};

export type ClimbProgress = {
  discipline: string;
  hardestSent: string;
  hardestSentRank: number;
  targetRank: number;
  rankGap: number;
  attempts30d: number;
  sessions30d: number;
};

export type GoalProgress = {
  goalId: string;
  activityType: GoalActivityType;
  targetValue: string;
  run?: RunProgress;
  climb?: ClimbProgress;
};
