import { describe, expect, it, vi } from "vitest";

// streamChatMessage reads from the fetch Response stream; the Node test
// environment provides ReadableStream, so we build a mock Response.
function sseResponse(frames: string[]): Response {
  const encoder = new TextEncoder();
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const frame of frames) {
        controller.enqueue(encoder.encode(frame));
      }
      controller.close();
    },
  });
  return new Response(stream, { status: 200, headers: { "content-type": "text/event-stream" } });
}

describe("streamChatMessage", () => {
  it("forwards deltas and resolves with the message id", async () => {
    const { streamChatMessage } = await import("./sse");

    const frames = [
      'event: delta\ndata: {"delta":"Hello"}\n\n',
      'event: delta\ndata: {"delta":" there"}\n\n',
      'event: done\ndata: {"messageId":"m-42"}\n\n',
    ];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(sseResponse(frames)),
    );

    const deltas: string[] = [];
    const messageId = await streamChatMessage("s1", "hi", (delta) => deltas.push(delta));

    expect(deltas).toEqual(["Hello", " there"]);
    expect(messageId).toBe("m-42");
    vi.unstubAllGlobals();
  });

  it("rejects on stream error events", async () => {
    const { streamChatMessage } = await import("./sse");

    const frames = ['event: error\ndata: {"error":"coach is unavailable"}\n\n'];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(sseResponse(frames)),
    );

    await expect(streamChatMessage("s1", "hi", () => {})).rejects.toThrow("coach is unavailable");
    vi.unstubAllGlobals();
  });
});
