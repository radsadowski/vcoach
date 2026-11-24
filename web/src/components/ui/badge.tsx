import { type HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

const variants = {
  default: "border-neutral-700 bg-neutral-800 text-neutral-200",
  success: "border-emerald-800 bg-emerald-950/60 text-emerald-300",
  warning: "border-amber-800 bg-amber-950/60 text-amber-300",
  danger: "border-red-900 bg-red-950/60 text-red-300",
  info: "border-sky-800 bg-sky-950/60 text-sky-300",
} as const;

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  variant?: keyof typeof variants;
};

export function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
