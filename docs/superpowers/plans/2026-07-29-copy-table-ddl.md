# Copy Table DDL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users copy the complete loaded table DDL script by pressing `c` in the DDL modal.

**Architecture:** `ddlModal` records whether a copy was requested and renders that confirmation in the footer. Its existing input handler will return Bubble Tea's OSC 52 clipboard command only for a non-empty, successfully loaded script.

**Tech Stack:** Go, Bubble Tea v2.0.8, Lip Gloss v2.0.5, Testify.

## Global Constraints

- Copy the complete original SQL string, never its sanitized, wrapped, or visible subset.
- Use `tea.SetClipboard`; add no dependencies or platform-specific subprocesses.
- `c` is a no-op while loading, after an error, or when the script is empty.
- Keep the modal open and preserve its scroll offset after copying.
- The footer advertises `c copy` and shows `Copied DDL` after copy is requested.
- Do not commit any changes.

---

## File structure

- `internal/app/ddl_modal.go`: state and footer rendering.
- `internal/app/update.go`: key routing and clipboard command.
- `internal/app/ddl_modal_test.go`: focused copy behavior tests.

### Task 1: Add the DDL copy command

**Files:**
- Modify: `internal/app/ddl_modal.go:9-53`
- Modify: `internal/app/update.go:731-755`
- Modify: `internal/app/ddl_modal_test.go`

**Interfaces:**
- Consumes: `tea.SetClipboard(s string) tea.Cmd`.
- Produces: an OSC 52 clipboard command from `(*Model).updateDDLModal`.

- [ ] **Step 1: Write a failing test for successful copying**

Add this test, importing `fmt`:

```go
func TestDDLModalCopiesOriginalSQL(t *testing.T) {
	model := Model{
		layout: newAppLayout(80, 24),
		ddlModal: &ddlModal{
			sql:    "CREATE TABLE public.\"Album\" (\n    \"Title\" varchar(160)\n);",
			offset: 1,
		},
	}

	updated, command := updateModel(t, model, keyPress('c', "c", 0))

	assert.NotNil(t, command)
	assert.Equal(t, model.ddlModal.sql, fmt.Sprint(command()))
	assert.True(t, updated.ddlModal.copied)
	assert.Equal(t, 1, updated.ddlModal.offset)
	assert.Contains(t, updated.ddlModal.view(updated.layout, "⠋"), "c copy")
	assert.Contains(t, updated.ddlModal.view(updated.layout, "⠋"), "Copied DDL")
}
```

- [ ] **Step 2: Verify the test fails**

Run `go test ./internal/app -run TestDDLModalCopiesOriginalSQL -count=1`.

Expected: compilation fails because `ddlModal.copied` does not exist and `c` returns no command.

- [ ] **Step 3: Add no-op regression coverage**

Import `errors` and add a table test that creates a model with each of: `loading: true`, a non-nil `err`, and empty SQL. For each, send `c`, then assert the command is nil and `updated.ddlModal.copied` is false.

- [ ] **Step 4: Implement the minimum behavior**

Add `copied bool` to `ddlModal`; reset it to false in `finish`. In `view`, construct this footer:

```go
footer := "↑/↓ scroll  •  PgUp/PgDown page  •  Home/End  •  c copy  •  Esc close"
if m.copied {
	footer += "  •  Copied DDL"
}
```

Then, before scroll branches in `updateDDLModal`, add:

```go
case keyMsg.String() == "c" && !m.ddlModal.loading && m.ddlModal.err == nil && m.ddlModal.sql != "":
	m.ddlModal.copied = true
	return tea.SetClipboard(m.ddlModal.sql)
```

- [ ] **Step 5: Verify and format**

Run `gofmt -w internal/app/ddl_modal.go internal/app/ddl_modal_test.go internal/app/update.go`, then `go test ./internal/app -count=1` and `go test ./... -count=1`.

Expected: formatting completes and all tests pass. Do not commit.
