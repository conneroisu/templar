---
id: task-113
title: Resolve circular dependencies in architecture
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
priority: high
---

## Description

Critical architectural issue with circular dependencies between build, registry, and server packages affecting maintainability and testing. This creates tight coupling and makes the system harder to test and modify.

## Acceptance Criteria

- [ ] Implement event bus pattern to break circular dependencies
- [ ] Remove direct dependencies between build and registry packages
- [ ] Ensure clean separation of concerns
- [ ] Add dependency graph validation to CI
- [ ] Verify improved testability after refactoring
