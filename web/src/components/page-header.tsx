import { Link } from '@tanstack/react-router'
import React from 'react'

type TabItem = { title: string; active?: boolean; path?: string; onClick?: () => void }
export type PageHeaderProps = { title: string; description?: string; tabs?: TabItem[]; actions?: React.ReactNode }

export function PageHeader(props: PageHeaderProps) {
  return (
    <div className="flex flex-1 flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h2 className="text-2xl font-semibold tracking-tight">{props.title}</h2>
          {props.description && <p className="text-muted-foreground">{props.description}</p>}
        </div>
        {props.actions && <div className="flex items-center gap-2">{props.actions}</div>}
      </div>
      {props.tabs && (
        <div className="flex flex-col gap-2">
          <div className="bg-muted text-muted-foreground inline-flex h-9 w-fit items-center justify-center rounded-lg p-[3px]">
            {props.tabs.map((tab, index) => (
              <React.Fragment key={index}>
                {tab.path ? (
                  <Link
                    to={tab.path}
                    data-state={tab.active ? 'active' : 'inactive'}
                    className="data-[state=active]:bg-background dark:data-[state=active]:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:outline-ring dark:data-[state=active]:border-input dark:data-[state=active]:bg-input/30 text-foreground dark:text-muted-foreground inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-2 py-1 text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:ring-[3px] focus-visible:outline-1 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:shadow-sm [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
                  >
                    {tab.title}
                  </Link>
                ) : (
                  <span
                    onClick={tab.onClick}
                    data-state={tab.active ? 'active' : 'inactive'}
                    className="cursor-pointer data-[state=active]:bg-background dark:data-[state=active]:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:outline-ring dark:data-[state=active]:border-input dark:data-[state=active]:bg-input/30 text-foreground dark:text-muted-foreground inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-2 py-1 text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:ring-[3px] focus-visible:outline-1 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:shadow-sm [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
                  >
                    {tab.title}
                  </span>
                )}
              </React.Fragment>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
