## Purpose

Define contextual Ctrl+G actions and safe renaming of the active saved connection.

## Requirements

### Requirement: Ctrl+G opens contextual actions
The application SHALL route Ctrl+G to an actions modal for the current database context instead of opening table DDL immediately.

#### Scenario: Active connection and selected table
- **WHEN** the user presses Ctrl+G while a saved connection is active and a table is selected
- **THEN** the modal offers an action to view DDL for the selected table and an action to rename the active saved connection

#### Scenario: Active connection without a selected table
- **WHEN** the user presses Ctrl+G while a saved connection is active but no table is selected
- **THEN** the modal keeps connection rename available and does not allow a DDL action

#### Scenario: Actions from raw query mode
- **WHEN** the user presses Ctrl+G from raw query mode with an active saved connection
- **THEN** the same contextual actions modal is opened

#### Scenario: Cancel action selection
- **WHEN** the actions modal is open and the user presses Esc
- **THEN** the modal closes without loading DDL or changing configuration

### Requirement: DDL remains available through actions
The application SHALL preserve the existing selected-table DDL behavior behind the DDL action.

#### Scenario: Select DDL action
- **WHEN** the user selects the DDL action for a selected table
- **THEN** the application closes the actions modal and opens the existing loading DDL modal for that table

#### Scenario: DDL is unavailable without a table
- **WHEN** no table is selected
- **THEN** the actions modal SHALL NOT initiate a DDL request

### Requirement: Active saved connection can be renamed
The application SHALL allow the user to change only the name of the active saved connection.

#### Scenario: Open rename input
- **WHEN** the user selects the rename connection action
- **THEN** the modal displays a focused text input prefilled with the active saved connection's current name

#### Scenario: Save a valid new name
- **WHEN** the user submits a non-empty name different from the current name
- **THEN** the application trims surrounding whitespace, saves that value to the active connection's `name` field, and leaves all other connection fields unchanged

#### Scenario: Empty name
- **WHEN** the user submits a name containing only whitespace
- **THEN** the application shows a validation error and does not write the configuration

#### Scenario: Unchanged name
- **WHEN** the user submits the active connection's existing name after trimming
- **THEN** the application performs no configuration write and leaves the saved connection unchanged

#### Scenario: Cancel rename
- **WHEN** the rename input is open and the user presses Esc
- **THEN** the application returns to action selection without changing configuration

#### Scenario: Config save fails
- **WHEN** persisting a valid renamed configuration fails
- **THEN** the application reports the error and retains the previous in-memory and on-disk connection name

#### Scenario: Rename succeeds
- **WHEN** persisting the renamed configuration succeeds
- **THEN** the application reports success and subsequent connection-list views display the new name

### Requirement: Rename does not affect the database session
Renaming a saved connection SHALL NOT reconnect, disconnect, or mutate the connected database.

#### Scenario: Rename while connected
- **WHEN** a saved connection is successfully renamed
- **THEN** the active database object, selected database, loaded tables, and connection settings remain unchanged

### Requirement: Active connection identity is unambiguous
The application SHALL identify the active saved connection by its position in the loaded configuration rather than by comparing connection settings.

#### Scenario: Duplicate connection settings
- **WHEN** multiple saved connections have identical engine and settings values
- **THEN** renaming changes only the entry that was selected to establish the active connection

#### Scenario: Earlier connection is deleted
- **WHEN** a saved connection before the active entry is removed
- **THEN** the application adjusts the active index so future renames still target the active entry

#### Scenario: Active connection is deleted
- **WHEN** the active saved connection is removed
- **THEN** the application clears its active saved-connection identity together with the current database session

### Requirement: Help describes the new action behavior
The application SHALL describe Ctrl+G as opening contextual actions rather than opening DDL directly.

#### Scenario: Footer help with an active connection
- **WHEN** the footer displays available commands for an active connection
- **THEN** the Ctrl+G help label describes actions and does not claim that Ctrl+G immediately opens table DDL
