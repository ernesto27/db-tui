# Requirement: Reopen the Last Used Connection

## Goal

When the TUI opens, automatically reconnect to the saved connection the user
last opened successfully. The user should arrive at their database without
choosing the same connection on every launch.

## Expected Behavior

1. After a saved connection opens successfully, remember it as the last used
   connection for future launches.
2. If the user successfully switches to another saved connection, remember the
   new connection instead.
3. On the next launch, automatically attempt to open the remembered connection.
4. If no connection has been remembered, show the usual disconnected starting
   screen.
5. A failed connection attempt must not replace the previously remembered
   connection.

## Unavailable Connections

If the remembered connection was removed or cannot be opened, show a clear
error message. Keep the TUI open and let the user choose or create another
connection. Do not repeatedly retry without user action.

If saving the new preference fails after a connection opens, keep that
connection available for the current session and tell the user that it may not
reopen automatically next time.

## Connection Identity and Privacy

Remember the specific saved connection, even when two saved connections have
the same name. Renaming or reordering saved connections must not cause a
different database to open automatically. Removing the remembered connection
must never redirect automatic startup to another connection.

Do not show passwords or connection strings in the remembered preference or in
error messages.

## Scope

This feature applies to TUI startup and saved connections. It adds no command
line option and does not change how users manually choose, create, edit, or
remove connections.

## Acceptance Criteria

- A saved connection used successfully is opened automatically on the next
  launch.
- Switching successfully to another saved connection changes the next launch
  target.
- Failed attempts leave the previous target unchanged.
- First launch without a remembered connection shows the usual starting screen.
- A missing or unavailable target produces an error while the TUI remains
  usable for choosing another connection.
- Duplicate names, renames, reordering, and deletion cannot silently open the
  wrong connection.
