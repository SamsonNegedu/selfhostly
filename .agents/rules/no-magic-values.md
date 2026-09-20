# No Magic Values

Replace magic numbers and strings with named constants that explain their meaning. This aligns with 12-factor app principles of explicit configuration.

## Examples

**❌ BAD - Magic values:**

```typescript
if (user.age > 18) { ... }
await setTimeout(5000);
connection.pool.max = 10;
language.code === 'de'
proficiency.order === 1
```

**✅ GOOD - Named constants:**

```typescript
const LEGAL_AGE = 18;
if (user.age > LEGAL_AGE) { ... }

const TIMEOUT_MS = 5000;
await setTimeout(TIMEOUT_MS);

const MAX_POOL_SIZE = 10;
connection.pool.max = MAX_POOL_SIZE;

import { DEFAULT_LANGUAGE } from '@/constants';
language.code === DEFAULT_LANGUAGE;

import { STARTING_PROFICIENCY_ORDER } from '@/constants';
proficiency.order === STARTING_PROFICIENCY_ORDER;
```

## Where to Place Constants

- **Cross-module:** `src/constants/` or relevant module's `constants/` folder
- **Single module:** Top of file or module-level constants file
- **Configuration:** Environment variables when value varies by environment

## When Magic Values are OK

- Array indices: `items[0]`, `slice(1)`
- Boolean literals: `true`, `false`
- Empty values: `''`, `0`, `null`
- Obvious math: `x * 2`, `percent / 100`
