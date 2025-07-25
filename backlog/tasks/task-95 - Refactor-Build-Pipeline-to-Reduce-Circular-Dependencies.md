---
id: task-95
title: Refactor Build Pipeline to Reduce Circular Dependencies
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

The BuildTask struct creates tight coupling between build and registry packages through direct *registry.ComponentInfo references, violating separation of concerns and making the code harder to test and maintain.

## Acceptance Criteria

- [ ] Create abstraction layer in internal/types/ for BuildableComponent interface
- [ ] Update BuildTask to use interface instead of concrete type
- [ ] Implement BuildCoordinator to separate responsibilities
- [ ] Update all references to use new abstraction
- [ ] Integration tests to verify functionality is preserved
