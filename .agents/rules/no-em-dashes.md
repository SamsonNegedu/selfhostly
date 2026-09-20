# No Em Dashes

Do not use the em dash character (Unicode U+2014) in code comments, docs, commit messages, PR text, or user-facing copy.

## Use instead

| Instead of                       | Use                                  |
| -------------------------------- | ------------------------------------ |
| `Clause [em dash] clause`        | `Clause. Clause` or `Clause: clause` |
| `Item [em dash] detail` (labels) | `Item: detail` or a comma            |
| Empty placeholder dash           | `'-'` or `n/a`                       |
| Parenthetical aside              | Commas or parentheses                |

## Examples

**Bad:**

```tsx
<p>Diagnostic result: not an official certification.</p>
```

**Good:**

```tsx
<p>Practice exam estimate. Not an official certification.</p>
```

**Bad:**

```typescript
/** Real exam session: assembled from content. */
```

**Good:**

```typescript
/** Real exam session assembled from content. */
```

## Scope

Applies to all prose this project authors: frontend copy, backend comments, ADRs, RFCs, Claude/Cursor rules, and skills. Existing third-party quoted text in source documents may keep original punctuation when it is a direct quotation.
