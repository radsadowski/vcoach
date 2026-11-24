export type ChatStreamEvent =
  | { event: "delta"; delta: string }
  | { event: "done"; messageId: string }
  | { event: "error"; error: string };

/**
 * Sends a chat message and streams the coach's reply.
 * Deltas are forwarded to onDelta as they arrive; resolves with the
 * persisted assistant message ID, or rejects on stream errors.
 */
export async function streamChatMessage(
  sessionId: string,
  content: string,
  onDelta: (delta: string) => void,
): Promise<string> {
  const res = await fetch(`/api/chat-sessions/${sessionId}/messages`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ content }),
  });
  if (!res.ok || !res.body) {
    throw new Error(`chat request failed: ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  const handleEvent = (event: string, data: string) => {
    if (event === "delta") {
      const payload = JSON.parse(data) as { delta: string };
      onDelta(payload.delta);
    } else if (event === "done") {
      const payload = JSON.parse(data) as { messageId: string };
      doneMessageId = payload.messageId;
    } else if (event === "error") {
      const payload = JSON.parse(data) as { error: string };
      throw new Error(payload.error);
    }
  };

  let doneMessageId = "";

  // Parse SSE frames: events are separated by a blank line; each frame has
  // "event: <name>" and "data: <json>" lines.
  const processBuffer = () => {
    for (;;) {
      const frameEnd = buffer.indexOf("\n\n");
      if (frameEnd === -1) return;
      const frame = buffer.slice(0, frameEnd);
      buffer = buffer.slice(frameEnd + 2);

      let eventName = "";
      let eventData = "";
      for (const line of frame.split("\n")) {
        if (line.startsWith("event: ")) eventName = line.slice(7).trim();
        if (line.startsWith("data: ")) eventData = line.slice(6).trim();
      }
      if (eventName && eventData) {
        handleEvent(eventName, eventData);
      }
    }
  };

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    processBuffer();
  }

  return doneMessageId;
}
