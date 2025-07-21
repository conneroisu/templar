---
id: task-122
title: Implement CSP nonce-based configuration for development
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
priority: high
---

## Description

High-priority security issue where development configuration allows overly permissive CSP directives including unsafe-eval and unsafe-inline, creating XSS vulnerability in development environments.

## Acceptance Criteria

- [ ] Implement nonce-based CSP even in development mode
- [ ] Remove unsafe-eval and unsafe-inline from development CSP
- [ ] Limit connect-src to specific origins instead of wildcard
- [ ] Maintain development functionality while improving security
- [ ] Add CSP testing for development configuration
