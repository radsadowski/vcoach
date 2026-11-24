import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { archivePlan, getPlan, listPlans } from "@vcoach/shared";
import { PlanView } from "./PlanView";

vi.mock("@vcoach/shared", () => ({
  archivePlan: vi.fn(),
  getPlan: vi.fn(),
  listPlans: vi.fn(),
}));

function renderPlan() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PlanView />
    </QueryClientProvider>,
  );
}

describe("PlanView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(archivePlan).mockResolvedValue();
  });

  it("prompts to ask the coach when no plan exists", async () => {
    vi.mocked(listPlans).mockResolvedValue([]);

    renderPlan();

    expect(await screen.findByText(/No active plan/i)).toBeInTheDocument();
  });

  it("renders the active plan with weeks, sessions and compliance badges", async () => {
    vi.mocked(listPlans).mockResolvedValue([
      { id: "p1", athleteId: "a1", goalId: null, version: 2, status: "active", title: "10K prep", notes: "", createdAt: "2026-11-01T10:00:00Z" },
      { id: "p0", athleteId: "a1", goalId: null, version: 1, status: "archived", title: "old", notes: "", createdAt: "2026-10-01T10:00:00Z" },
    ]);
    vi.mocked(getPlan).mockResolvedValue({
      id: "p1",
      athleteId: "a1",
      goalId: null,
      version: 2,
      status: "active",
      title: "10K prep",
      notes: "",
      createdAt: "2026-11-01T10:00:00Z",
      weeks: [
        {
          id: "w1",
          planId: "p1",
          weekNumber: 1,
          startDate: "2026-11-23",
          focus: "base",
          sessions: [
            {
              id: "s1",
              weekId: "w1",
              athleteId: "a1",
              sport: "run",
              sessionType: "easy",
              description: "easy run",
              plannedDate: "2026-11-24",
              targetDurationMin: 40,
              targetDistanceKm: null,
              targetIntensity: "zone 2",
              status: "completed",
              matchedActivityId: "act1",
              completedAt: "2026-11-24T07:00:00Z",
            },
            {
              id: "s2",
              weekId: "w1",
              athleteId: "a1",
              sport: "run",
              sessionType: "long",
              description: "",
              plannedDate: "2026-11-28",
              targetDurationMin: null,
              targetDistanceKm: 12,
              targetIntensity: null,
              status: "missed",
              matchedActivityId: null,
              completedAt: null,
            },
          ],
        },
      ],
    });

    renderPlan();

    expect(await screen.findByText("10K prep")).toBeInTheDocument();
    expect(screen.getByText(/Week 1/)).toBeInTheDocument();
    expect(screen.getByText("completed")).toBeInTheDocument();
    expect(screen.getByText("missed")).toBeInTheDocument();
    // history shows the archived version
    expect(screen.getByText(/History/)).toBeInTheDocument();
  });
});
