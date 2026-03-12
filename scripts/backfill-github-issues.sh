#!/usr/bin/env bash
# One-time backfill: import existing tasks and ideas as GitHub Issues.
# Usage: bash scripts/backfill-github-issues.sh [--dry-run]
#
# Creates GitHub Issues for all tasks in done.txt, to-do.txt, progressing.txt
# and all ideas in ideas.txt. Writes GitHub: #NNN back to each block.
#
# WARNING: This creates many issues (190+). Run with --dry-run first!

set -euo pipefail

CONFIG=".claude/github-issues.json"
REPO="$(jq -r '.repo' "$CONFIG")"
DRY_RUN=false
DELAY=1  # seconds between API calls

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=true
  echo "=== DRY RUN MODE — no issues will be created ==="
fi

# Section letter → label mapping (read from config)
section_label() {
  local letter="$1"
  jq -r ".labels.sections.\"$letter\" // \"\"" "$CONFIG"
}

# Priority → label mapping
priority_label() {
  local priority="$1"
  jq -r ".labels.priority.\"$priority\" // \"\"" "$CONFIG"
}

# Determine which section a task falls under based on its line number in a file.
# Usage: get_section_letter <file> <line_number>
get_section_letter() {
  local file="$1" line="$2"
  local result=""
  while IFS=: read -r sline _; do
    if (( sline <= line )); then
      # Extract the section letter from the header line
      local letter
      letter=$(sed -n "${sline}p" "$file" | grep -oE 'SEZIONE ([A-Z])' | awk '{print $2}')
      if [[ -n "$letter" ]]; then
        result="$letter"
      fi
    fi
  done < <(grep -n 'SEZIONE [A-Z]' "$file" 2>/dev/null)
  echo "$result"
}

# Parse a task block starting at a given line.
# Extracts: code, title, priority, full body text.
# Usage: parse_task_block <file> <start_line>
# Outputs lines: CODE, TITLE, PRIORITY, SECTION_LETTER, BODY (rest is body until next separator)
create_task_issue() {
  local file="$1" status_label="$2"
  local current_code="" current_title="" current_priority="" current_body="" in_block=false
  local line_num=0 block_start=0

  while IFS= read -r line; do
    (( line_num++ )) || true

    # Detect task header: [x] CODE-NNN — Title  or  [ ] CODE-NNN — Title  or  [~] CODE-NNN — Title
    if [[ "$line" =~ ^\[.\]\ ([A-Z][A-Z0-9]+-[0-9]{3})\ —\ (.+)$ ]]; then
      # If we had a previous block, flush it
      if [[ -n "$current_code" ]]; then
        flush_task "$file" "$current_code" "$current_title" "$current_priority" "$current_body" "$status_label" "$block_start"
      fi
      current_code="${BASH_REMATCH[1]}"
      current_title="${BASH_REMATCH[2]}"
      current_priority=""
      current_body=""
      in_block=true
      block_start=$line_num
      continue
    fi

    if $in_block; then
      # Extract priority
      if [[ "$line" =~ ^[[:space:]]*Priorita:[[:space:]]*(.+)$ ]]; then
        current_priority="${BASH_REMATCH[1]}"
      fi
      # Accumulate body (skip separator lines)
      if [[ "$line" != "------"* ]]; then
        current_body+="$line"$'\n'
      fi
    fi
  done < "$file"

  # Flush the last block
  if [[ -n "$current_code" ]]; then
    flush_task "$file" "$current_code" "$current_title" "$current_priority" "$current_body" "$status_label" "$block_start"
  fi
}

CREATED=0
SKIPPED=0
ERRORS=0

flush_task() {
  local file="$1" code="$2" title="$3" priority="$4" body="$5" status_label="$6" block_start="$7"

  # Check if issue already exists
  local existing
  existing=$(gh issue list --repo "$REPO" --search "[$code] in:title" --label task --json number --jq '.[0].number' 2>/dev/null || true)
  if [[ -n "$existing" ]]; then
    echo "  SKIP [$code] — already exists as #$existing"
    (( SKIPPED++ )) || true
    return
  fi

  # Determine section
  local section_letter
  section_letter=$(get_section_letter "$file" "$block_start")
  local sec_label
  sec_label=$(section_label "$section_letter")

  # Determine priority label
  local pri_label
  pri_label=$(priority_label "$priority")

  # Build labels
  local labels="claude-code,task,$status_label"
  [[ -n "$pri_label" ]] && labels+=",$pri_label"
  [[ -n "$sec_label" ]] && labels+=",$sec_label"

  # Build issue body (truncate to 65000 chars to avoid API limits)
  local issue_body
  issue_body=$(cat <<EOF
**Code:** $code | **Priority:** ${priority:-N/A} | **Section:** ${section_letter:-N/A}

## Task Details

$(echo "$body" | head -200)

---
*Backfilled by Claude Code*
EOF
  )
  issue_body="${issue_body:0:65000}"

  if $DRY_RUN; then
    echo "  WOULD CREATE [$code] $title (labels: $labels)"
    (( CREATED++ )) || true
    return
  fi

  local issue_url
  issue_url=$(gh issue create --repo "$REPO" \
    --title "[$code] $title" \
    --body "$issue_body" \
    --label "$labels" 2>&1) || {
    echo "  ERROR [$code] — gh issue create failed: $issue_url"
    (( ERRORS++ )) || true
    return
  }

  local issue_num
  issue_num=$(echo "$issue_url" | grep -oE '[0-9]+$')
  echo "  CREATED [$code] → #$issue_num"
  (( CREATED++ )) || true

  # Close the issue if it's a done task
  if [[ "$status_label" == "status:done" ]]; then
    gh issue close "$issue_num" --repo "$REPO" --comment "Task completed (backfilled from done.txt)." 2>/dev/null || true
  fi

  sleep "$DELAY"
}

# --- Process ideas ---
create_idea_issue() {
  local file="$1" close_after="${2:-false}"
  local current_code="" current_title="" current_category="" current_body="" in_block=false

  while IFS= read -r line; do
    # Detect idea header: IDEA-NNN — Title
    if [[ "$line" =~ ^(IDEA-[0-9]{3})\ —\ (.+)$ ]]; then
      if [[ -n "$current_code" ]]; then
        flush_idea "$current_code" "$current_title" "$current_category" "$current_body" "$close_after"
      fi
      current_code="${BASH_REMATCH[1]}"
      current_title="${BASH_REMATCH[2]}"
      current_category=""
      current_body=""
      in_block=true
      continue
    fi

    if $in_block; then
      if [[ "$line" =~ ^[[:space:]]*Categoria:[[:space:]]*(.+)$ ]]; then
        current_category="${BASH_REMATCH[1]}"
      fi
      if [[ "$line" != "------"* ]]; then
        current_body+="$line"$'\n'
      fi
    fi
  done < "$file"

  if [[ -n "$current_code" ]]; then
    flush_idea "$current_code" "$current_title" "$current_category" "$current_body" "$close_after"
  fi
}

flush_idea() {
  local code="$1" title="$2" category="$3" body="$4" close_after="$5"

  local existing
  existing=$(gh issue list --repo "$REPO" --search "[$code] in:title" --label idea --json number --jq '.[0].number' 2>/dev/null || true)
  if [[ -n "$existing" ]]; then
    echo "  SKIP [$code] — already exists as #$existing"
    (( SKIPPED++ )) || true
    return
  fi

  local labels="claude-code,idea"

  local issue_body
  issue_body=$(cat <<EOF
**Category:** ${category:-N/A}

## Idea Details

$(echo "$body" | head -100)

---
*Backfilled by Claude Code*
EOF
  )
  issue_body="${issue_body:0:65000}"

  if $DRY_RUN; then
    echo "  WOULD CREATE [$code] $title (labels: $labels)"
    (( CREATED++ )) || true
    return
  fi

  local issue_url
  issue_url=$(gh issue create --repo "$REPO" \
    --title "[$code] $title" \
    --body "$issue_body" \
    --label "$labels" 2>&1) || {
    echo "  ERROR [$code] — gh issue create failed: $issue_url"
    (( ERRORS++ )) || true
    return
  }

  local issue_num
  issue_num=$(echo "$issue_url" | grep -oE '[0-9]+$')
  echo "  CREATED [$code] → #$issue_num"
  (( CREATED++ )) || true

  if [[ "$close_after" == "true" ]]; then
    gh issue close "$issue_num" --repo "$REPO" --reason "not planned" --comment "Idea was disapproved (backfilled)." 2>/dev/null || true
  fi

  sleep "$DELAY"
}

# ============================================================
# EXECUTION ORDER: done first, then todo, then progressing,
# then ideas, then disapproved ideas
# ============================================================

echo ""
echo "=== Processing done.txt (completed tasks) ==="
if [[ -f done.txt ]]; then
  create_task_issue "done.txt" "status:done"
else
  echo "  File not found, skipping."
fi

echo ""
echo "=== Processing to-do.txt (pending tasks) ==="
if [[ -f to-do.txt ]]; then
  create_task_issue "to-do.txt" "status:todo"
else
  echo "  File not found, skipping."
fi

echo ""
echo "=== Processing progressing.txt (in-progress tasks) ==="
if [[ -f progressing.txt ]]; then
  create_task_issue "progressing.txt" "status:in-progress"
else
  echo "  File not found, skipping."
fi

echo ""
echo "=== Processing ideas.txt (active ideas) ==="
if [[ -f ideas.txt ]]; then
  create_idea_issue "ideas.txt" "false"
else
  echo "  File not found, skipping."
fi

echo ""
echo "=== Processing idea-disapproved.txt (rejected ideas) ==="
if [[ -f idea-disapproved.txt ]]; then
  create_idea_issue "idea-disapproved.txt" "true"
else
  echo "  File not found, skipping."
fi

echo ""
echo "============================================"
echo "Backfill complete!"
echo "  Created: $CREATED"
echo "  Skipped: $SKIPPED"
echo "  Errors:  $ERRORS"
echo "============================================"
