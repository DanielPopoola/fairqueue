import * as React from "react"
import { cn } from "./Button"
import { motion } from "motion/react"

interface ProgressProps extends React.HTMLAttributes<HTMLDivElement> {
  value: number
}

const Progress = React.forwardRef<HTMLDivElement, ProgressProps>(
  ({ className, value, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "relative h-2 w-full overflow-hidden rounded-full bg-[#E5E7EB]",
        className
      )}
      {...props}
    >
      <motion.div
        className="h-full w-full flex-1 bg-[var(--color-primary)] transition-all"
        initial={{ x: "-100%" }}
        animate={{ x: `-${100 - (value || 0)}%` }}
        transition={{ duration: 0.5, ease: "easeInOut" }}
      />
    </div>
  )
)
Progress.displayName = "Progress"

export { Progress }
