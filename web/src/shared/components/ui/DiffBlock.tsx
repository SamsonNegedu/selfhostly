import * as React from "react"
import { diffLines } from "diff"

import { CodeBlock, type CodeLine } from "./CodeBlock"

interface DiffBlockProps {
  before: string
  after: string
  "aria-label"?: string
  className?: string
}

// Shows the full file with removed lines in red and added lines in green.
function DiffBlock({ before, after, ...props }: DiffBlockProps) {
  const lines = React.useMemo<CodeLine[]>(
    () =>
      diffLines(before, after).flatMap((part) =>
        part.value
          .replace(/\n$/, "")
          .split("\n")
          .map((text) => ({ text, mark: part.added ? "add" : part.removed ? "del" : undefined }))
      ),
    [before, after]
  )

  return <CodeBlock lines={lines} aria-label={props["aria-label"] ?? "Changes"} className={props.className} />
}

export { DiffBlock }
