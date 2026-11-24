import ReactMarkdown from "react-markdown";
import { cn } from "../../lib/utils";

type MessageBubbleProps = {
  message: {
    role: "user" | "assistant";
    content: string;
    streaming?: boolean;
  };
};

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === "user";

  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[80%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed",
          isUser
            ? "bg-neutral-100 text-neutral-900"
            : "bg-neutral-800/80 text-neutral-100",
          message.streaming && "animate-pulse",
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="prose prose-invert prose-sm max-w-none prose-p:my-1 prose-li:my-0.5 prose-strong:text-neutral-100 prose-code:text-neutral-200">
            <ReactMarkdown>{message.content}</ReactMarkdown>
          </div>
        )}
      </div>
    </div>
  );
}
