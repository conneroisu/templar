---
id: task-63
title: Create central interfaces package
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

Missing key interfaces like ComponentRegistry, BuildPipeline, and FileWatcher causes tight coupling and difficult testing. Need central interfaces to improve architecture.

## Acceptance Criteria

- [ ] Create internal/interfaces package with core interfaces
- [ ] Define ComponentRegistry interface
- [ ] Add BuildPipeline interface for testability
- [ ] Create FileWatcher interface to reduce coupling
- [ ] Update packages to use interfaces instead of concrete types
