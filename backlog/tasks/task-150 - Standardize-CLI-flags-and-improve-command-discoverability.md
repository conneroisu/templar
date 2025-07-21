---
id: task-150
title: Standardize CLI flags and improve command discoverability
status: To Do
assignee: []
created_date: '2025-07-20'
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
