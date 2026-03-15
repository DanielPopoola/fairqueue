import * as React from "react"
import { cn } from "./Button"

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "success" | "warning" | "error" | "info"
}

const badgeVariants = ({ variant = "default", className = "" }: { variant?: string, className?: string }) => {
  return cn(
    "inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
    {
      "bg-gray-100 text-gray-800": variant === "default",
      "bg-green-100 text-green-800": variant === "success",
      "bg-amber-100 text-amber-800": variant === "warning",
      "bg-red-100 text-red-800": variant === "error",
      "bg-blue-100 text-blue-800": variant === "info",
    },
    className
  )
}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={badgeVariants({ variant, className })} {...props} />
  )
}

export { Badge, badgeVariants }
