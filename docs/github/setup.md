# GitHub Setup

`servicekit` manages repo files, but GitHub merge policy and branch protection live in repository settings. Apply them manually with `gh` after the repository exists on GitHub.

## Requirements

- `gh` authenticated with repository admin access
- the `fracturing-space/game` repository already created on GitHub

## Standard Policy

The default Fracturing.Space policy is:

- allow only squash merges
- disable merge commits
- disable rebase merges
- protect `main`
- require pull requests for changes to `main`
- enforce the rule for admins too
- block force-pushes and branch deletion on `main`

## Commands

```bash
gh repo edit fracturing-space/game \
  --enable-squash-merge \
  --enable-merge-commit=false \
  --enable-rebase-merge=false

gh api -X PUT repos/fracturing-space/game/branches/main/protection --input - <<'JSON'
{
  "required_status_checks": null,
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": false,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0,
    "require_last_push_approval": false,
    "dismissal_restrictions": {
      "users": [],
      "teams": [],
      "apps": []
    },
    "bypass_pull_request_allowances": {
      "users": [],
      "teams": [],
      "apps": []
    }
  },
  "restrictions": null,
  "required_linear_history": false,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": false,
  "lock_branch": false,
  "allow_fork_syncing": false
}
JSON
```

## Verify

```bash
gh api repos/fracturing-space/game --jq '{default_branch: .default_branch, allow_squash_merge: .allow_squash_merge, allow_merge_commit: .allow_merge_commit, allow_rebase_merge: .allow_rebase_merge}'

gh api repos/fracturing-space/game/branches/main/protection --jq '{enforce_admins: .enforce_admins.enabled, required_pull_request_reviews: .required_pull_request_reviews.required_approving_review_count, allow_force_pushes: .allow_force_pushes.enabled, allow_deletions: .allow_deletions.enabled}'
```

## Notes

- This does not require a PR approval count; it only forces changes onto `main` through PRs.
- If org-level rulesets already exist, they may override or conflict with this repo-level setup.
- Once `servicekit` grows a dedicated GitHub setup command, it should emit the same policy by default.
