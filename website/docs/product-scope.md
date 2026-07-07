# JudgeHub Frontend Product Scope

## Purpose

This document defines the intended frontend product scope for JudgeHub during the early planning and UI foundation phases. It exists to keep future implementation work aligned and to prevent scope drift.

## Product Overview

JudgeHub is a modern Online Judge platform with two clearly separated frontend areas:

- User App
- Admin App

Both areas may share a visual language, but they must not share the same navigation structure or application shell.

## Who The Product Serves

- User App: students, contestants, self-learners, and community members who solve problems, join contests, review submissions, and track their progress.
- Admin App: platform operators, contest managers, problem setters, moderators, and system administrators who manage the platform and oversee operations.

## User App Scope

The User App is the participant-facing product area. It should feel clean, modern, welcoming, and focused on discovery, participation, and progress tracking.

### Primary User Routes

- `/`
- `/problems`
- `/contests`
- `/submissions`
- `/ranking`
- `/discuss`
- `/profile`

### User Navigation Rules

- Use a horizontal top navigation.
- Include: Problems, Contests, Submissions, Ranking, Discuss.
- Never include Manage, Admin, or any administration-focused tab.
- The right side of the navbar may contain:
  - Search
  - Theme toggle
  - Notifications
  - User avatar or account dropdown

### User App Functional Themes

- Browse and filter problems
- View contests and contest states
- Track submission history
- Check rankings and competitive standing
- Read and join discussions
- View personal profile and progress

## Admin App Scope

The Admin App is a separate platform-management area. It is not an extension of the user navbar and must live under its own route and shell.

### Primary Admin Routes

- `/admin`
- `/admin/users`
- `/admin/problems`
- `/admin/contests`
- `/admin/submissions`
- `/admin/leaderboard`
- `/admin/groups`
- `/admin/system-logs`
- `/admin/settings`

### Admin Navigation Rules

- Use a vertical left sidebar.
- Include: Dashboard, Users, Problems, Contests, Submissions, Leaderboard, Groups, System Logs, Settings.
- Do not reuse the user top navigation.

### Admin App Functional Themes

- Platform monitoring
- User management
- Problem and contest management
- Submission moderation and oversight
- Leaderboard and grouping administration
- System visibility and configuration

## Explicit Non-Goals For This Phase

- No real backend integration
- No real authentication
- No real role permissions
- No payment features
- No real-time judge status
- No production monitoring
- No production API client layer
- No large-scale page implementation
- No dependency expansion without a documented need

## MVP Frontend Scope

- Define frontend architecture and route boundaries
- Create separate user and admin layout plans
- Create a shared design direction for future implementation
- Prepare mock-data-driven page implementation phases
- Establish agent workflow, review rules, and scope guardrails

## Current Out-Of-Scope Items

- Real authentication and session management
- Authorization and role-enforcement logic
- Payments, billing, and subscriptions
- WebSocket or polling-based live judge updates
- Production analytics and observability dashboards
- Full backend API integration
- Production deployment concerns

## Early Data Strategy

- Use mock data only.
- Keep mocks under `website/lib/mocks`.
- Shape mock data around realistic future entities, but avoid speculative abstraction.

## UX Direction

- Desktop-first structure
- Responsive behavior still required
- Spacious layouts with strong hierarchy
- Avoid dense enterprise-style screens
- Preserve a modern SaaS dashboard feel

## Scope Guardrails

- Separate shells for User App and Admin App are mandatory.
- Routing, navigation, and layout rules take priority over convenience.
- Shared visual language is allowed; shared information architecture is not.
- New pages, features, or flows should be added only if they clearly belong to one of the two defined app areas.
