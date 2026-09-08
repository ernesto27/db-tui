---
name: update-readme
description: Review commits since README.md was last changed and update it with verified user-facing features, CLI commands, setup changes, and shortcuts.
---

# Update README

Keep `README.md` aligned with shipped, user-visible behavior.

## Establish the review range

1. Confirm the repository has a committed `README.md` history.
2. Find its latest committed change:

   ```sh
   git log -1 --format=%H -- README.md
   ```

3. Review commits from that commit through `HEAD`, excluding the baseline itself.
4. If README has never been committed, review the repository history or ask the user for a starting point.
5. Honor an explicit user-provided commit range or baseline instead.

Check the working tree first. Preserve uncommitted README changes and never discard them.

## Gather evidence

- Inspect relevant commit diffs and the resulting implementation, tests, CLI help, configuration examples, and user-visible strings.
- Consolidate related commits into one documented capability.
- Include only shipped, user-facing behavior. Exclude refactors, test-only work, CI-only changes, and incomplete work.
- Do not infer behavior from commit messages alone.
- Do not describe an engine-specific capability as universal.

## Update README

Edit only sections that need correction, preserving the existing style.

- Update supported engines, installation, setup, and configuration requirements.
- Add verified capabilities to `Features`.
- Add or correct discoverable shortcuts in `Keyboard reference`.
- Add non-interactive CLI commands with copyable examples, required flags, output behavior, and engine-specific limitations.
- Remove or correct claims contradicted by the current implementation.

Use safe placeholders in examples; never include credentials or local data.

## Approval and verification

Before editing `README.md`, show the reviewed range, verified changes, target sections, and intentional omissions. Wait for explicit approval.

After approval, re-read the updated README for accuracy and run the repository's relevant validation. Report changed sections and verification.
