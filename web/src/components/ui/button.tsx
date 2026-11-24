import { type ButtonHTMLAttributes, forwardRef } from "react";
import { cn } from "../../lib/utils";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "outline" | "ghost" | "destructive";
  size?: "default" | "sm";
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 disabled:pointer-events-none disabled:opacity-50",
        variant === "default" && "bg-neutral-100 text-neutral-900 hover:bg-neutral-200",
        variant === "outline" && "border border-neutral-700 bg-transparent text-neutral-200 hover:bg-neutral-800",
        variant === "ghost" && "text-neutral-300 hover:bg-neutral-800 hover:text-neutral-100",
        variant === "destructive" && "bg-red-900/40 text-red-200 hover:bg-red-900/60",
        size === "sm" && "h-8 px-3",
        size === "default" && "h-9 px-4",
        className,
      )}
      {...props}
    />
  ),
);
Button.displayName = "Button";
