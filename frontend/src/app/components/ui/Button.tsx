import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const buttonVariants = ({ variant = "primary", size = "default", className = "" }: { variant?: "primary" | "secondary" | "danger" | "ghost", size?: "default" | "sm" | "lg", className?: string }) => {
  return cn(
    "inline-flex items-center justify-center whitespace-nowrap rounded-lg text-sm font-semibold ring-offset-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-primary)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
    {
      "bg-[var(--color-primary)] text-white hover:bg-blue-700": variant === "primary",
      "bg-white text-[var(--color-primary)] border border-[var(--color-primary)] hover:bg-blue-50": variant === "secondary",
      "bg-[var(--color-error)] text-white hover:bg-red-600": variant === "danger",
      "hover:bg-gray-100 text-gray-700": variant === "ghost",
      "h-10 px-4 py-2": size === "default",
      "h-8 rounded-md px-3 text-xs": size === "sm",
      "h-12 rounded-lg px-8 text-base": size === "lg",
    },
    className
  )
}

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean
  variant?: "primary" | "secondary" | "danger" | "ghost"
  size?: "default" | "sm" | "lg"
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={buttonVariants({ variant, size, className })}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
