---
id: task-153
title: Eliminate adapter anti-pattern architectural debt
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies:
  - task-63
  - task-113
  - task-131
---

## Description

Dave (Architecture Agent) identified CRITICAL architectural anti-pattern in adapters that violates Interface Segregation Principle and creates dangerous tight coupling. Adapters contain logic that swallows errors and hide interface mismatches.

## Acceptance Criteria

- [ ] Remove all adapter types from adapters.go
- [ ] Redesign interfaces to match concrete implementations
- [ ] Update concrete types to implement interfaces directly
- [ ] Remove adapter instantiation from DI container
- [ ] Eliminate circular dependencies
- [ ] Improve testability and mocking capabilities
