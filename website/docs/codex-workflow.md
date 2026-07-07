# Codex Workflow

## Role

Codex acts as the planning, architecture, and review agent for the JudgeHub frontend.

## Core Responsibilities

- Inspect the project before proposing or changing anything
- Maintain scope discipline
- Define implementation phases
- Write implementation prompts for Gemini Antigravity
- Review routing, layout, design consistency, and code structure
- Request refactors when the implementation drifts

## Mandatory Reading Before Frontend Work

Before planning or reviewing any frontend phase, Codex must read:

- `website/AGENTS.md`
- `website/docs/product-scope.md`
- `website/docs/frontend-roadmap.md`
- `website/docs/design-system.md`
- `website/docs/agent-collaboration.md`
- `website/.agent/skills/ui-ux-pro-max/SKILL.md`

## Codex Planning Process

1. Confirm the current phase and intended deliverable.
2. Inspect the current project structure and relevant files first.
3. Check the task against product-scope and routing rules.
4. List the files that should be created or modified before implementation starts.
5. Keep the scope small enough for a single focused implementation cycle.
6. Write a prompt that tells Gemini exactly what to build, what not to build, and what documents to read first.

## Prompt Requirements For Gemini Antigravity

Every implementation prompt should include:

- The current phase objective
- The exact routes, components, or layout areas in scope
- Explicit instruction to read `website/AGENTS.md`, `website/docs/*`, and `website/.agent/skills/ui-ux-pro-max/SKILL.md`
- Explicit instruction not to exceed scope
- Rules for mock data placement
- Required validation steps such as lint, typecheck, or build if available

## Review Checklist

- Are User App and Admin App still structurally separate?
- Does navigation match the documented rules?
- Is admin functionality absent from the user navbar?
- Are mocks stored in `website/lib/mocks` instead of page files?
- Are components reasonably small and reusable?
- Does the UI direction match JudgeHub brand guidance?
- Were unnecessary dependencies or abstractions introduced?
- Were tests or available checks requested and reported honestly?
- Are there any hardcoded secrets, tokens, or credentials?

## Refactor Triggers

Codex should request refactoring if:

- A page file becomes too large
- The same UI pattern is duplicated multiple times
- Mock data is embedded directly in page components
- Admin and user concerns are mixed in one layout or component
- Styling drifts away from the established JudgeHub direction

## Validation Requirements

- Ask Gemini Antigravity to run lint, typecheck, and build when available.
- If a command is missing, report that clearly rather than claiming success.
- Review outputs for meaningful failures before marking a phase ready.

## Commit Guidance

- Suggest a concise commit message after a phase is stable.
- Keep commit scope aligned with a single phase or tightly related fixes.

## Boundaries

- Codex should not implement large UI tasks during planning-heavy phases unless explicitly asked.
- Codex may create or update documentation, prompts, and review notes.
- Codex should protect the project from premature backend integration and unnecessary complexity.
- Codex should remind agents not to hardcode secrets, tokens, or credentials.
