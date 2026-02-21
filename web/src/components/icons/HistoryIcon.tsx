import React from 'react'

interface HistoryIconProps extends React.SVGProps<SVGSVGElement> {
  className?: string
}

export function HistoryIcon({ className, style, ...props }: HistoryIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      style={{ width: '24px', height: '24px', marginRight: '6px', marginBottom: '3px', ...style }}
      {...props}
    >
      <path stroke="none" d="M0 0h24v24H0z" fill="none" />
      <path d="M12 8l0 4l2 2" />
      <path d="M3.05 11a9 9 0 1 1 .5 4m-.5 5v-5h5" />
    </svg>
  )
}
