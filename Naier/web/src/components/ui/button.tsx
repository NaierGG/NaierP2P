import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded-full text-sm font-semibold transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 disabled:shadow-none",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground shadow-[0_14px_30px_rgba(54,176,224,0.28)] hover:-translate-y-0.5 hover:bg-primary/95 hover:shadow-[0_18px_40px_rgba(54,176,224,0.35)]",
        destructive: "bg-destructive text-destructive-foreground shadow-[0_12px_28px_rgba(190,34,62,0.22)] hover:bg-destructive/90",
        outline: "border border-border/80 bg-card/70 text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.03)] hover:border-primary/35 hover:bg-accent/80 hover:text-foreground",
        secondary: "border border-border/70 bg-secondary/70 text-secondary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.03)] hover:bg-secondary",
        ghost: "text-muted-foreground hover:bg-accent/80 hover:text-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-5 py-2.5",
        sm: "h-8 px-3.5 text-xs",
        lg: "h-12 px-6 text-base",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
