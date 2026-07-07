# JudgeHub Frontend Design System

## Brand Direction

- Brand name: JudgeHub
- Product feeling: modern SaaS dashboard for competitive programming
- Visual tone: clean, spacious, capable, and polished
- Primary color family: purple

## Color Palette

- Primary: purple as the main brand accent
- Background: white or very light lavender
- Surfaces: white cards and panels
- Neutrals: cool grays for text, borders, and muted UI
- Success, warning, error, and info colors should be restrained and consistent
- Avoid introducing random secondary accents without a reason tied to product meaning

## Core Visual Principles

- Favor clarity over decoration.
- Keep layouts breathable and structured.
- Use white or very light lavender backgrounds.
- Use white cards with subtle separation from the page.
- Avoid random accent colors and noisy visual effects.
- Keep hierarchy obvious through spacing, type scale, and surface treatment.

## Layout Direction

- Desktop-first
- Responsive and mobile-safe
- Wide containers for data-heavy screens
- Clean gutters and generous section spacing
- Strong separation between page chrome and content regions

## Spacing Rules

- Favor generous spacing over compact density
- Use consistent section rhythm between page header, filters, content, and secondary panels
- Keep card padding comfortable for both text and data display
- Avoid stacking too many small controls without breathing room
- The UI must not feel dense

## Navigation Patterns

### User App

- Horizontal top navigation
- Product-browsing and participation orientation
- Right utility cluster for search, theme toggle, notifications, and user menu

### Admin App

- Vertical left sidebar
- Operational and management orientation
- Dashboard-first structure with predictable section grouping

## Surface System

- App background: white or soft lavender tint
- Primary surfaces: white
- Borders: thin and low-contrast but visible
- Corners: rounded, modern, not overly soft
- Shadows: subtle and restrained

## Card Rules

- Cards should use white backgrounds, rounded corners, thin borders, and subtle shadows
- Card headers should establish clear hierarchy without excessive decoration
- Card content should avoid cramped spacing
- Use cards to group meaningfully related information, not as a default wrapper for everything

## Button Rules

- Primary buttons should use the JudgeHub purple accent
- Secondary buttons should rely on neutral surfaces and clear borders
- Button sizes should be consistent across pages
- Hover, focus, and disabled states must be obvious
- Avoid excessive gradients, glow, or oversized shadows

## Badge And Status Rules

- Use compact, readable badges for difficulty, status, and category labels
- Status colors should follow a stable semantic mapping
- Badges should support quick scanning without overpowering the page
- Avoid using too many badge colors within one table or card group

## Table Rules

- Tables should prioritize scanability and alignment
- Row height must remain comfortable, not overly compressed
- Use clear headers, muted separators, and restrained zebra or hover treatments
- Sticky headers or overflow handling may be used when justified by content length

## Form And Filter Rules

- Filters should be grouped logically and placed near the content they affect
- Inputs should remain clean, neutral, and easy to scan
- Forms should use clear labels, helpful placeholders, and visible validation states
- Avoid oversized filter bars that dominate the page before the main content

## Typography Direction

- Modern and readable
- Light visual character without sacrificing clarity
- Strong heading hierarchy
- Comfortable line length for dense data and descriptive text
- Avoid playful or decorative typography unless explicitly requested
- Typography should feel thin, modern, and readable

## Color Rules

- Purple is the primary brand color.
- Neutral grays should support readability and information density.
- Status colors should be used sparingly and consistently.
- Do not introduce unrelated accent palettes without a documented reason.

## Navigation Rules

- User App navigation must be horizontal and top-aligned
- Admin App navigation must be vertical and sidebar-based
- Active states should be clear but not loud
- Utility controls should not crowd primary navigation items

## Chart And Dashboard Card Rules

- Dashboard charts should be simple, readable, and decision-oriented
- Cards that summarize metrics should highlight one key number or trend at a time
- Do not overload one dashboard row with too many competing metrics
- Charts should use the JudgeHub palette, with purple as the primary highlight

## Empty State Rules

- Empty states should clearly explain what is missing and what the user can do next
- Keep illustrations or decoration restrained
- Use actionable copy where appropriate, especially in admin management screens

## Loading And Skeleton Rules

- Prefer lightweight skeletons or loading blocks that match final layout shape
- Avoid flashy loading effects
- Loading UI should preserve structure and reduce layout shift

## Component Behavior Guidelines

- Components should feel light, intentional, and stable.
- Hover and focus states must be visible without creating layout shift.
- Tables, filters, cards, and lists should prioritize scanability.
- Modals and drawers should be used carefully and not become the default solution for everything.

## Accessibility Baseline

- Visible keyboard focus
- Sufficient text contrast
- Semantic headings and landmarks
- Click targets large enough for regular use
- Icons never as the only source of meaning when text is needed

## Responsive Direction

- Desktop-first remains the planning default
- Tablet and mobile layouts should preserve usability rather than duplicate desktop density
- Collapse secondary controls before compromising content readability
- User navbar and admin sidebar should adapt differently based on their separate app roles

## Admin Vs User Visual Relationship

- Both apps should feel like part of the same JudgeHub brand family
- User App should feel more participant-focused and discovery-oriented
- Admin App should feel more operational, structured, and information-driven
- Visual language can be shared, but shells, navigation, and page framing must remain distinct

## Design Anti-Patterns

- Dense dashboards with minimal spacing
- Random color usage across cards and charts
- Heavy shadow stacks
- Mixing user and admin navigation models
- Repeating large blocks of near-identical card UI without extracting a pattern
- Shipping pages that look functional but visually unfinished

## Relationship To The UI Skill

`website/.agent/skills/ui-ux-pro-max/SKILL.md` is the required design intelligence source for implementation work. When Gemini Antigravity builds UI, the skill should be used together with this document:

- This document defines JudgeHub-specific direction.
- The skill provides broader UI/UX tactics, references, and implementation heuristics.
- If the skill suggests patterns that conflict with this document, JudgeHub project rules take priority unless explicitly changed.
