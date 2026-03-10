"use client"

import * as React from "react"
import { cn } from "@/lib/utils"
import { ChevronDown } from "lucide-react"

const Select = ({ children, defaultValue, onValueChange }: any) => {
  const [value, setValue] = React.useState(defaultValue)

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newVal = e.target.value
    setValue(newVal)
    if (onValueChange) onValueChange(newVal)
  }

  // 为子组件提供上下文或直接处理
  return (
    <div className="relative w-full group">
      <select
        defaultValue={defaultValue}
        onChange={handleChange}
        className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
      >
        {React.Children.map(children, (child) => {
          if (child.type === SelectContent) {
            return child.props.children
          }
          return null
        })}
      </select>
      {React.Children.map(children, (child) => {
        if (child.type === SelectTrigger) {
          return React.cloneElement(child, { value })
        }
        return null
      })}
    </div>
  )
}

const SelectTrigger = ({ className, children, value, ...props }: any) => (
  <div
    className={cn(
      "flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
      className
    )}
    {...props}
  >
    {value || <span className="text-muted-foreground">请选择...</span>}
    <ChevronDown className="h-4 w-4 opacity-50" />
  </div>
)

const SelectValue = ({ placeholder }: any) => null // 仅占位

const SelectContent = ({ children }: any) => children

const SelectItem = ({ value, children }: any) => (
  <option value={value}>{children}</option>
)

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectItem,
}

const SelectGroup = ({ children }: any) => <>{children}</>
