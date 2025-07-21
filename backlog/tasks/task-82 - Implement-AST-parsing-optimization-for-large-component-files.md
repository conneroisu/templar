---
id: task-82
title: Implement AST parsing optimization for large component files
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

Performance analysis identified AST parsing as a CPU bottleneck in scanner operations, especially for large .templ files. Current implementation blocks worker threads during parsing, reducing overall throughput.

## Acceptance Criteria

- [ ] AST parsing caching mechanism implemented
- [ ] Large file parsing performance improved by 50%
- [ ] Worker thread blocking eliminated
- [ ] Memory usage remains within bounds during parsing
