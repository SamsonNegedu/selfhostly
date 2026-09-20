// The diff package ships without types. This covers the one function the app uses.
declare module 'diff' {
    export interface Change {
        value: string
        added?: boolean
        removed?: boolean
        count?: number
    }

    export function diffLines(oldStr: string, newStr: string): Change[]
}
