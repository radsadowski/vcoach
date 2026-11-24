import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createChatSession, listChatMessages, listChatSessions, streamChatMessage } from "@vcoach/shared";
import { Button } from "../../components/ui/button";
import { cn } from "../../lib/utils";
import { MessageBubble } from "./MessageBubble";

type DisplayMessage = {
  role: "user" | "assistant";
  content: string;
  streaming?: boolean;
};

export function ChatView() {
  const { sessionId } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [messages, setMessages] = useState<DisplayMessage[]>([]);
  const [input, setInput] = useState("");
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const sessions = useQuery({ queryKey: ["chat-sessions"], queryFn: listChatSessions });

  const history = useQuery({
    queryKey: ["chat-messages", sessionId],
    queryFn: () => listChatMessages(sessionId!),
    enabled: !!sessionId,
  });

  useEffect(() => {
    if (history.data) {
      setMessages(history.data.map((message) => ({ role: message.role, content: message.content })));
    }
  }, [history.data]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const send = useMutation({
    mutationFn: async (content: string) => {
      let id = sessionId;
      if (!id) {
        const session = await createChatSession(content.slice(0, 60));
        id = session.id;
        navigate(`/chat/${id}`, { replace: true });
      }
      await streamChatMessage(id!, content, (delta) => {
        setMessages((current) => {
          const next = [...current];
          const last = next[next.length - 1];
          if (last?.role === "assistant" && last.streaming) {
            next[next.length - 1] = { ...last, content: last.content + delta };
          } else {
            next.push({ role: "assistant", content: delta, streaming: true });
          }
          return next;
        });
      });
    },
    onMutate: (content) => {
      setError(null);
      setMessages((current) => [...current, { role: "user", content }]);
    },
    onSuccess: () => {
      setMessages((current) =>
        current.map((message) => (message.streaming ? { ...message, streaming: false } : message)),
      );
      queryClient.invalidateQueries({ queryKey: ["chat-sessions"] });
      queryClient.invalidateQueries();
    },
    onError: (err) => {
      setMessages((current) => current.filter((message) => !message.streaming));
      setError(err instanceof Error ? err.message : "chat failed");
    },
  });

  const submit = () => {
    const content = input.trim();
    if (!content || send.isPending) return;
    setInput("");
    send.mutate(content);
  };

  return (
    <div className="flex h-full">
      <div className="flex w-56 shrink-0 flex-col border-r border-neutral-800 bg-neutral-900/40 p-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setMessages([]);
            navigate("/chat");
          }}
        >
          New chat
        </Button>
        <div className="mt-3 flex flex-col gap-1 overflow-y-auto">
          {sessions.data?.map((session) => (
            <button
              key={session.id}
              onClick={() => navigate(`/chat/${session.id}`)}
              className={cn(
                "truncate rounded-md px-3 py-2 text-left text-sm",
                session.id === sessionId
                  ? "bg-neutral-800 text-neutral-100"
                  : "text-neutral-400 hover:bg-neutral-800/60 hover:text-neutral-200",
              )}
            >
              {session.title}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-1 flex-col">
        <div className="flex-1 space-y-4 overflow-y-auto p-6">
          {messages.length === 0 && (
            <div className="mt-16 text-center text-sm text-neutral-500">
              Ask the coach for a goal and a training plan —{" "}
              <span className="text-neutral-400">"I want to run a 10K in 12 weeks."</span>
            </div>
          )}
          {messages.map((message, index) => (
            <MessageBubble key={index} message={message} />
          ))}
          {error && (
            <div className="mx-auto max-w-lg rounded-md border border-red-900 bg-red-950/40 p-3 text-sm text-red-300">
              {error}
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        <div className="border-t border-neutral-800 p-4">
          <div className="mx-auto flex max-w-3xl gap-2">
            <textarea
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  submit();
                }
              }}
              placeholder="Message the coach…"
              rows={2}
              className="flex-1 resize-none rounded-md border border-neutral-700 bg-neutral-900 px-3 py-2 text-sm text-neutral-100 placeholder:text-neutral-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-500"
            />
            <Button onClick={submit} disabled={send.isPending || !input.trim()}>
              {send.isPending ? "…" : "Send"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
