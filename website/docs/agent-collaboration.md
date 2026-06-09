# Agent Collaboration Guide

## Purpose

This document defines how Codex and Gemini Antigravity collaborate on the JudgeHub frontend without overlapping responsibilities or drifting from scope.

## Role Split

### Codex

- Planner
- Architect
- Reviewer

Codex owns:

- Phase definition
- Architecture rules
- Prompt writing
- Code review
- Refactor requests
- Scope control

### Gemini Antigravity

- Frontend Implementer
- UI Builder

Gemini owns:

- Layout implementation
- Component building
- Typography, spacing, and polish
- Responsive adjustments
- UI bug fixing
- Running available frontend checks

## Shared Source Of Truth

Both agents must treat the following as required references:

- `website/AGENTS.md`
- `website/docs/product-scope.md`
- `website/docs/frontend-roadmap.md`
- `website/docs/design-system.md`
- `website/docs/codex-workflow.md`
- `website/.agent/skills/ui-ux-pro-max/SKILL.md`

## Required Collaboration Loop

1. Codex reads the source-of-truth documents.
2. Codex defines the current phase and writes a focused implementation prompt.
3. Gemini Antigravity reads the same documents and the UI skill before coding.
4. Gemini Antigravity implements only the prompt scope.
5. Gemini Antigravity runs available validation checks.
6. Codex reviews the implementation for architecture, design fidelity, quality, and risk.
7. Gemini Antigravity refactors or fixes review findings.
8. Commit only after the phase is stable.

## Prompt Template For Codex Planning

Use this structure when Codex prepares a UI implementation task:

1. Phase name and goal
2. Files to read first
3. Exact routes, layouts, and components in scope
4. Files expected to be created or modified
5. Rules that must not be violated
6. Validation commands to run if available
7. Explicit out-of-scope list

## Prompt Template For Gemini Antigravity Implementation

Use this structure when Gemini receives a UI task:

1. Read `website/AGENTS.md`, `website/docs/*`, and `website/.agent/skills/ui-ux-pro-max/SKILL.md` first
2. Restate the phase goal
3. Implement only the listed routes, components, and mocks
4. Keep user and admin shells separate
5. Keep mock data in `website/lib/mocks`
6. Run lint, typecheck, and build if available
7. Report any missing command clearly
8. Do not add dependencies or backend integration unless explicitly requested

## Prompt Template For Codex Review

Use this structure when Codex reviews Gemini output:

1. Summary of reviewed scope
2. Findings ordered by severity
3. Architecture and routing compliance check
4. Design-system and UI-skill compliance check
5. Validation results and gaps
6. Required refactors before commit

## Handoff Rules

- Prompts should be small, specific, and phase-bound.
- Review feedback should be concrete and tied to documented rules.
- UI decisions should reference the design-system document or the UI skill, not personal preference alone.
- If a conflict exists between convenience and architecture, architecture wins.

## Escalation Rules

- If implementation pressure suggests mixing user and admin shells, stop and refactor.
- If new dependencies appear necessary, document why before adding them.
- If backend integration is requested early, isolate the request and confirm that it belongs to the current phase.

## Scope Drift Prevention Rules

- Do not implement features outside the active phase.
- Do not silently add routes that were not planned.
- Do not turn mock-driven work into backend integration work.
- Do not blend admin controls into user-facing components for convenience.

## Commit Rules After Each Phase

- Commit only after implementation and review are complete.
- Prefer one commit per stable phase or tightly related fix set.
- Use commit messages that reflect the phase deliverable clearly.

## Done Criteria For A Phase

A phase is ready to close when:

- The requested scope is implemented and no more
- Routing and layout rules are preserved
- Mock data placement is correct
- Available checks pass
- Review findings are addressed or clearly documented
