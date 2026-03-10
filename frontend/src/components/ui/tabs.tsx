"use client"

import * as React from "react"
import { cn } from "@/lib/utils"

const Tabs = ({ children, defaultValue, onValueChange, className }: any) => {
  const [value, setValue] = React.useState(defaultValue)

  const handleValueChange = (newVal: string) => {
    setValue(newVal)
    if (onValueChange) onValueChange(newVal)
  }

  return (
    <div className={cn("w-full", className)}>
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child as React.ReactElement<any>, { 
            activeValue: value, 
            onValueChange: handleValueChange 
          })
        }
        return child
      })}
    </div>
  )
}

const TabsList = ({ children, className, activeValue, onValueChange }: any) => (
  <div className={cn("inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground", className)}>
    {React.Children.map(children, (child) => {
      if (React.isValidElement(child)) {
        return React.cloneElement(child as React.ReactElement<any>, { 
          isActive: child.props.value === activeValue,
          onClick: () => onValueChange(child.props.value)
        })
      }
      return child
    })}
  </div>
)

const TabsTrigger = ({ children, className, isActive, onClick }: any) => (
  <button
    onClick={onClick}
    className={cn(
      "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
      isActive ? "bg-background text-foreground shadow-sm" : "hover:bg-background/50 hover:text-foreground",
      className
    )}
  >
    {children}
  </button>
)

const TabsContent = ({ children, value, activeValue, className }: any) => {
  if (value !== activeValue) return null
  return <div className={cn("mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2", className)}>{children}</div>
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
