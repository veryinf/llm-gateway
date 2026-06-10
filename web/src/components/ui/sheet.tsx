import * as React from "react"
import { cn } from "@/lib/utils"

interface SheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
  side?: "left" | "right"
}

function Sheet({ open, onOpenChange, children }: SheetProps) {
  return (
    <>
      {open && (
        <div className="fixed inset-0 z-50">
          <div
            className="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity"
            onClick={() => onOpenChange(false)}
          />
          {children}
        </div>
      )}
    </>
  )
}

const SheetHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div className={cn("flex flex-col space-y-2 text-center sm:text-left", className)} {...props} />
)

const SheetTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h2 ref={ref} className={cn("text-lg font-semibold text-foreground", className)} {...props} />
  )
)
SheetTitle.displayName = "SheetTitle"

const SheetDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...props }, ref) => (
    <p ref={ref} className={cn("text-sm text-muted-foreground", className)} {...props} />
  )
)
SheetDescription.displayName = "SheetDescription"

function SheetContent({
  className,
  side = "left",
  style,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { side?: "left" | "right" }) {
  return (
    <div
      style={style}
      className={cn(
        "fixed top-0 z-50 h-full w-80 bg-background p-6 shadow-lg transition-transform duration-300 ease-in-out",
        side === "left" ? "left-0" : "right-0",
        className
      )}
      {...props}
    />
  )
}

export { Sheet, SheetHeader, SheetTitle, SheetDescription, SheetContent }
