---
id: task-112
title: Fix build queue silent failures under load
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
priority: high
---

## Description

High-priority reliability issue where build requests are silently dropped when queues are full, causing silent failures without user notification. This affects system reliability under load.

## Acceptance Criteria

- [ ] Replace silent queue dropping with backpressure handling
- [ ] Implement retry logic or priority-based queue management
- [ ] Add proper error reporting for dropped requests
- [ ] Ensure no build requests are lost silently
- [ ] Add monitoring for queue overflow scenarios
