import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { createChatSession, listChatMessages, listChatSessions, streamChatMessage } from "@vcoach/shared";
import { ChatView } from "./ChatView";

vi.mock("@vcoach/shared", () => ({
  createChatSession: vi.fn(),
  listChatMessages: vi.fn(),
  listChatSessions: vi.fn().mockResolvedValue([]),
  streamChatMessage: vi.fn(),
}));

function renderChat(initialEntry = "/chat") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/chat/:sessionId" element={<ChatView />} />
          <Route path="/chat" element={<ChatView />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ChatView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listChatSessions).mockResolvedValue([]);
    window.history.replaceState({}, "", "/chat");
  });

  it("shows history messages of the opened session", async () => {
    vi.mocked(listChatMessages).mockResolvedValue([
      {
        id: "1",
        chatSessionId: "s1",
        athleteId: "a1",
        role: "user",
        content: "hello coach",
        toolCalls: null,
        createdAt: "2026-11-23T10:00:00Z",
      },
      {
        id: "2",
        chatSessionId: "s1",
        athleteId: "a1",
        role: "assistant",
        content: "hello athlete",
        toolCalls: null,
        createdAt: "2026-11-23T10:00:05Z",
      },
    ]);
    renderChat("/chat/s1");

    expect(await screen.findByText("hello coach")).toBeInTheDocument();
    expect(screen.getByText("hello athlete")).toBeInTheDocument();
  });

  it("streams the coach reply as it arrives", async () => {
    const session = { id: "s1", athleteId: "a1", title: "new", createdAt: "", updatedAt: "" };
    vi.mocked(createChatSession).mockResolvedValue(session);
    vi.mocked(streamChatMessage).mockImplementation(async (_id, _content, onDelta) => {
      onDelta("Plan ");
      onDelta("is ready");
      return "2";
    });
    renderChat();

    const input = screen.getByPlaceholderText("Message the coach…");
    await userEvent.type(input, "make me a plan");
    await userEvent.keyboard("{Enter}");

    await waitFor(() => {
      expect(screen.getByText(/Plan is ready/)).toBeInTheDocument();
    });
    expect(screen.getByText("make me a plan")).toBeInTheDocument();
  });

  it("surfaces stream errors", async () => {
    vi.mocked(createChatSession).mockResolvedValue({
      id: "s1",
      athleteId: "a1",
      title: "new",
      createdAt: "",
      updatedAt: "",
    });
    vi.mocked(streamChatMessage).mockRejectedValue(new Error("coach is unavailable"));
    renderChat();

    const input = screen.getByPlaceholderText("Message the coach…");
    await userEvent.type(input, "hello");
    await userEvent.keyboard("{Enter}");

    await waitFor(() => {
      expect(screen.getByText(/coach is unavailable/)).toBeInTheDocument();
    });
  });
});
