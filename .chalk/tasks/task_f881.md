---
id: task_f881
title: Web: Accessibility — keyboard navigation, focus management, ARIA
type: task
status: open
priority: 3
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:53Z
updated_at: 2026-06-11T22:28:13Z
---
Ensure the web app meets baseline accessibility requirements: keyboard navigability, visible focus states, semantic HTML, and screen-reader compatibility.

WHAT TO DO:
- All interactive elements (nav items, article cards, buttons, inputs) are reachable and activatable via keyboard (Tab, Enter/Space).
- Modals (search, add-link) trap focus when open; Escape closes them.
- Visible focus ring on all focusable elements (using the theme's primary/hover colors or a standard outline).
- Semantic HTML: nav, main, article, button, a, h1/h2/h3 used appropriately; article list items are li elements or have role='listitem'.
- ARIA labels on icon-only buttons (favorite, archive, delete, vote).
- Sufficient color contrast (the token palette is designed to be high-contrast; verify with DevTools).
- Reader supports browser zoom (text reflows correctly at 200% zoom).

VERIFICATION (agent-testable):
1. Tab through the sidebar: all three primary nav items (Read, Explore, You) receive visible focus and can be activated with Enter.
2. In the reading list, Tab moves focus to article cards; pressing Enter on a focused card opens the reader.
3. Opening the search modal and pressing Tab cycles through the search input and results only; Escape closes the modal.
4. Icon-only buttons (favorite, vote) have aria-label attributes visible in DevTools.
5. Zooming the browser to 200% on the reading list and article reader: text reflows and no content is clipped or inaccessible.
