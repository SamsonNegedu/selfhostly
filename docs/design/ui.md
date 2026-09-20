# Web UI

The web UI follows a Claude design canvas: dark-first slate theme with a light theme, one indigo
accent, a status color system, shared primitives and every screen rebuilt. The canvas is the design reference:
https://claude.ai/artifact/5yDejXYjqJ18JuW87UR9xx (private, so it has to be shared by its owner).

## Where things are

- Design tokens: `web/src/styles/globals.css`, mapped in `web/tailwind.config.js`.
- Primitives: `web/src/shared/components/ui/`. Open `/dev/ui` in the dev server to see all of them in both themes.
- Screens: `web/src/features/` (one directory per screen).
- Rules for working on it: `.agents/rules/frontend-ui.md`.
- Run it with sample data: `docs/dev-fixtures/README.md`.
- Backend work the designs needed and what is still missing: [backend-gaps.md](backend-gaps.md).

## Decisions worth knowing

- Dark is the default. Light is a first-class theme, not an afterthought.
- Color means something. Status colors (green, amber, red, blue, grey) are only for state, and always come with
  a word. The indigo accent is for primary actions, the active tab, selected cards and the focus ring.
- One page per job, one card level. Dialogs and sheets are flat.
- Phones get their own layouts: a bottom tab bar, compact Fleet rows, a sticky action bar on the app page.
  Tablets (768 to 1023px) get the icon rail so the page has room.
- Where the API has nothing behind a design, the UI says so and does not fake it. Examples: no metrics history
  (charts show readings taken while the page is open), no environment store (variables live in the compose
  file), no alerts or backup.

## Not verified against real Docker

Live container metrics, streaming deploy logs, Save and redeploy, Restore version, Retry start and real
tunnels need real containers, which the fixtures do not start. Check them on a real node after changing them.
