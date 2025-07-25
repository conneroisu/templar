---
id: task-150
title: Standardize CLI flags and improve command discoverability
status: In Progress
assignee: []
created_date: '2025-07-20'
updated_date: '2025-07-22'
labels: []
dependencies:
  - task-75
  - task-84
  - task-91
---

## Description

Carol (UX Agent) identified HIGH-severity usability issues with inconsistent flag naming patterns and poor command structure. Developers struggle with non-intuitive commands like --no-open vs --no-browser and lack workflow guidance.

## Acceptance Criteria

- [ ] Implement consistent flag naming with aliases
- [ ] Add interactive templar command for workflow guidance
- [ ] Provide templar tutorial command
- [ ] All flags have short aliases where appropriate
- [ ] Flag validation suggests similar alternatives
- [ ] Help text includes practical examples

## Implementation Plan

1. Audit current CLI flags across all commands for inconsistencies
2. Design consistent flag naming conventions and short aliases
3. Implement standardized flag validation with suggestions
4. Add interactive 'templar' command for workflow guidance
5. Create 'templar tutorial' command for learning workflows
6. Update help text with practical examples and common use cases
7. Add command aliases and shortcuts for discoverability
8. Test flag consistency and validation across all commands
