---
inclusion: always
---

# Context7 Usage

Use the Context7 MCP power to look up current documentation for libraries and frameworks before writing or recommending code. This ensures answers reflect the latest API surfaces rather than stale training data.

## When to Use Context7

- Before writing code that uses a third-party library (e.g., `gorilla/websocket`, `go-sql-driver/mysql`, `fast-check`, `pinia`, `vueuse`)
- When referencing Go standard library APIs that may have changed in Go 1.22+
- When answering questions about Vue 3 Composition API, Vite config, or Vitest usage
- During spec creation or task execution that involves library integration

## When NOT to Use Context7

- For project-internal code patterns already documented in other steering files
- For simple language syntax or well-established conventions
- When the answer is already clear from the existing codebase
