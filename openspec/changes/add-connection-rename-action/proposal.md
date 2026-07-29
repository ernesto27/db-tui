## Why

Saved connections receive generated names that cannot currently be changed without editing `config.json` manually. Users need an in-app way to give the active saved connection a recognizable label while preserving the existing Ctrl+G table DDL workflow.

## What Changes

- Change Ctrl+G from opening table DDL directly to opening an actions modal.
- Offer explicit actions to view DDL for the selected table or rename the active saved connection.
- Add a rename input that validates the new label and persists only the connection's `name` field.
- Keep the current database session open and leave its engine and connection settings unchanged.
- Track the active saved connection by config index so entries with identical settings are renamed correctly.
- Report config-save failures without changing the in-memory connection name.

## Capabilities

### New Capabilities

- `connection-actions`: Defines Ctrl+G action selection and safe renaming of the active saved connection.

### Modified Capabilities

None.

## Impact

- `internal/app/`: Ctrl+G routing, modal state and view rendering, active saved-connection identity, config-save command lifecycle, footer help, and automated tests.
- `internal/config/`: Existing `Config.Save` behavior is reused; config schema remains unchanged.
- No database adapter, SQL, connection credentials, or external dependency changes are required.
