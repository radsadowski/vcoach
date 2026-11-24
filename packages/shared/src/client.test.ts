import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, apiFetch, getHealth } from "./client";

describe("apiFetch", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("decodes successful JSON responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ status: "ok" }), { status: 200 }),
      ),
    );

    const health = await getHealth();
    expect(health.status).toBe("ok");
  });

  it("throws ApiError with status and body on failures", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(`{"error":"invalid grade"}`, { status: 400 }),
      ),
    );

    try {
      await apiFetch("/api/goals");
      expect.unreachable("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      const apiError = err as ApiError;
      expect(apiError.status).toBe(400);
      expect(apiError.body).toContain("invalid grade");
    }
  });
});
