#!/usr/bin/env python3
"""Codebase memory builder for the agentic fleet pipeline.

Generates a structured Markdown summary of a project's codebase,
intended as context for Claude Code agents during automated idea scouting.

Zero external dependencies — stdlib only.
"""

import argparse
import fnmatch
import os
import sys
from pathlib import Path


# ── Constants ───────────────────────────────────────────────────────────────

DEFAULT_MAX_DEPTH = 4
DEFAULT_MAX_SIZE = 40000  # 40KB cap for output
KEY_FILE_PREVIEW_LINES = 50

# Files that are always skipped in the tree
ALWAYS_SKIP = {
    ".git", "__pycache__", "node_modules", ".venv", "venv",
    ".tox", ".mypy_cache", ".pytest_cache", ".ruff_cache",
    "dist", "build", ".next", ".nuxt", "target",
    ".DS_Store", "Thumbs.db",
}

# Package manifest files to look for
PACKAGE_MANIFESTS = [
    "package.json", "pyproject.toml", "Cargo.toml", "go.mod",
    "pom.xml", "build.gradle", "Gemfile", "composer.json",
    "mix.exs", "pubspec.yaml",
]

# Task/idea files
TASK_FILES = ["to-do.txt", "progressing.txt", "done.txt"]
IDEA_FILES = ["ideas.txt", "idea-disapproved.txt"]


# ── Gitignore Parsing ──────────────────────────────────────────────────────

def load_gitignore_patterns(root: Path) -> list[str]:
    """Load .gitignore patterns from the project root."""
    gitignore = root / ".gitignore"
    if not gitignore.exists():
        return []
    patterns = []
    try:
        for line in gitignore.read_text(encoding="utf-8", errors="replace").splitlines():
            line = line.strip()
            if line and not line.startswith("#"):
                patterns.append(line)
    except OSError:
        pass
    return patterns


def is_ignored(rel_path: str, patterns: list[str]) -> bool:
    """Check if a relative path matches any gitignore pattern."""
    name = os.path.basename(rel_path)
    for pattern in patterns:
        # Match against the full relative path and the basename
        if fnmatch.fnmatch(name, pattern) or fnmatch.fnmatch(rel_path, pattern):
            return True
        # Handle directory patterns (trailing /)
        if pattern.endswith("/"):
            dir_pattern = pattern.rstrip("/")
            if fnmatch.fnmatch(name, dir_pattern) or fnmatch.fnmatch(rel_path, dir_pattern):
                return True
    return False


# ── Tree Builder ───────────────────────────────────────────────────────────

def build_tree(root: Path, max_depth: int, gitignore_patterns: list[str]) -> str:
    """Build a filtered directory tree as a string."""
    lines = []

    def _walk(directory: Path, prefix: str, depth: int):
        if depth > max_depth:
            return
        try:
            entries = sorted(directory.iterdir(), key=lambda e: (not e.is_dir(), e.name.lower()))
        except PermissionError:
            return

        # Filter entries
        filtered = []
        for entry in entries:
            if entry.name in ALWAYS_SKIP:
                continue
            rel = str(entry.relative_to(root))
            if is_ignored(rel, gitignore_patterns):
                continue
            filtered.append(entry)

        for i, entry in enumerate(filtered):
            is_last = i == len(filtered) - 1
            connector = "└── " if is_last else "├── "
            suffix = "/" if entry.is_dir() else ""
            lines.append(f"{prefix}{connector}{entry.name}{suffix}")
            if entry.is_dir():
                extension = "    " if is_last else "│   "
                _walk(entry, prefix + extension, depth + 1)

    lines.append(f"{root.name}/")
    _walk(root, "", 1)
    return "\n".join(lines)


# ── Statistics ─────────────────────────────────────────────────────────────

def compute_stats(root: Path, gitignore_patterns: list[str]) -> dict:
    """Compute file statistics for the project."""
    ext_counts: dict[str, int] = {}
    total_files = 0
    total_loc = 0

    for dirpath, dirnames, filenames in os.walk(root):
        # Prune ignored directories in-place
        dirnames[:] = [
            d for d in dirnames
            if d not in ALWAYS_SKIP
            and not is_ignored(str(Path(dirpath, d).relative_to(root)), gitignore_patterns)
        ]

        for fname in filenames:
            fpath = Path(dirpath) / fname
            rel = str(fpath.relative_to(root))
            if is_ignored(rel, gitignore_patterns):
                continue

            total_files += 1
            ext = fpath.suffix.lower() if fpath.suffix else "(no ext)"
            ext_counts[ext] = ext_counts.get(ext, 0) + 1

            # Estimate LOC for text files
            try:
                if fpath.stat().st_size < 500_000:  # Skip very large files
                    total_loc += len(fpath.read_bytes().split(b"\n"))
            except (OSError, UnicodeDecodeError):
                pass

    return {
        "total_files": total_files,
        "total_loc": total_loc,
        "by_extension": dict(sorted(ext_counts.items(), key=lambda x: -x[1])),
    }


# ── Key File Previews ─────────────────────────────────────────────────────

def read_preview(filepath: Path, max_lines: int = KEY_FILE_PREVIEW_LINES) -> str:
    """Read the first N lines of a file."""
    if not filepath.exists():
        return ""
    try:
        lines = filepath.read_text(encoding="utf-8", errors="replace").splitlines()
        preview = lines[:max_lines]
        result = "\n".join(preview)
        if len(lines) > max_lines:
            result += f"\n... ({len(lines) - max_lines} more lines)"
        return result
    except OSError:
        return "(unreadable)"


def extract_architecture_section(claude_md: Path) -> str:
    """Extract the ## Architecture section from CLAUDE.md."""
    if not claude_md.exists():
        return "(no CLAUDE.md found)"
    try:
        content = claude_md.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return "(unreadable)"

    lines = content.splitlines()
    in_section = False
    section_lines = []

    for line in lines:
        if line.strip().startswith("## Architecture"):
            in_section = True
            continue
        if in_section:
            if line.strip().startswith("## ") or line.strip().startswith("<!-- CTDF:"):
                break
            section_lines.append(line)

    text = "\n".join(section_lines).strip()
    return text if text else "(Architecture section is empty)"


# ── Backlog Summary ───────────────────────────────────────────────────────

def summarize_backlog(root: Path) -> str:
    """Summarize task and idea files."""
    lines = []
    has_local_files = False

    # Ideas
    ideas_path = root / "ideas.txt"
    if ideas_path.exists():
        has_local_files = True
        ideas = _extract_titles(ideas_path, prefix="IDEA-")
        lines.append(f"### Existing Ideas ({len(ideas)} total)")
        if ideas:
            for code, title in ideas:
                lines.append(f"- {code} — {title}")
        else:
            lines.append("(none)")
        lines.append("")

    # Tasks by status
    for label, fname in [("To Do", "to-do.txt"), ("In Progress", "progressing.txt"), ("Done", "done.txt")]:
        fpath = root / fname
        if fpath.exists():
            has_local_files = True
            tasks = _extract_titles(fpath)
            lines.append(f"### Tasks — {label} ({len(tasks)} total)")
            if tasks:
                for code, title in tasks:
                    lines.append(f"- {code} — {title}")
            else:
                lines.append("(none)")
            lines.append("")

    if not has_local_files:
        lines.append("_No local task/idea files found. If using platform-only mode,")
        lines.append("task and idea data should be fetched via the platform CLI at runtime._")
        lines.append("")

    return "\n".join(lines)


def _extract_titles(filepath: Path, prefix: str = "") -> list[tuple[str, str]]:
    """Extract (code, title) pairs from a task/idea file."""
    import re
    results = []
    try:
        content = filepath.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return results

    # Match task headers: [x] CODE-0001 — Title
    task_re = re.compile(r"^\[.\]\s+([A-Z]{3,5}-\d{4})\s+—\s+(.+)$", re.MULTILINE)
    # Match idea headers: IDEA-CODE-0001 — Title
    idea_re = re.compile(r"^(IDEA-[A-Z]{3,5}-\d{4})\s+—\s+(.+)$", re.MULTILINE)

    for m in task_re.finditer(content):
        results.append((m.group(1), m.group(2).strip()))
    for m in idea_re.finditer(content):
        results.append((m.group(1), m.group(2).strip()))

    return results


# ── Main Generator ─────────────────────────────────────────────────────────

def generate_memory(root: Path, max_depth: int, max_size: int) -> str:
    """Generate the full project memory Markdown document."""
    root = root.resolve()
    gitignore_patterns = load_gitignore_patterns(root)

    sections = []

    # Header
    sections.append("# Project Memory — Codebase Summary\n")
    sections.append(f"_Generated for: `{root.name}`_\n")

    # Project Overview
    sections.append("## Project Overview\n")
    sections.append(extract_architecture_section(root / "CLAUDE.md"))
    sections.append("")

    # File Tree
    sections.append("## File Tree\n")
    sections.append("```")
    sections.append(build_tree(root, max_depth, gitignore_patterns))
    sections.append("```")
    sections.append("")

    # Statistics
    stats = compute_stats(root, gitignore_patterns)
    sections.append("## Statistics\n")
    sections.append(f"- **Total files:** {stats['total_files']}")
    sections.append(f"- **Estimated LOC:** ~{stats['total_loc']:,}")
    sections.append("- **By extension:**")
    for ext, count in list(stats["by_extension"].items())[:20]:
        sections.append(f"  - `{ext}`: {count}")
    sections.append("")

    # Key Files
    sections.append("## Key Files\n")

    # README
    for readme_name in ["README.md", "README.rst", "README.txt", "README"]:
        readme = root / readme_name
        if readme.exists():
            sections.append(f"### {readme_name}\n")
            sections.append("```")
            sections.append(read_preview(readme))
            sections.append("```")
            sections.append("")
            break

    # Package manifests
    for manifest in PACKAGE_MANIFESTS:
        mpath = root / manifest
        if mpath.exists():
            sections.append(f"### {manifest}\n")
            sections.append("```")
            sections.append(read_preview(mpath, 30))
            sections.append("```")
            sections.append("")

    # CLAUDE.md (full)
    claude_md = root / "CLAUDE.md"
    if claude_md.exists():
        sections.append("### CLAUDE.md\n")
        sections.append("```")
        sections.append(read_preview(claude_md, 200))
        sections.append("```")
        sections.append("")

    # Backlog Summary
    sections.append("## Current Backlog Summary\n")
    sections.append(summarize_backlog(root))

    output = "\n".join(sections)

    # Enforce size cap
    if len(output.encode("utf-8")) > max_size:
        # Truncate and add note
        while len(output.encode("utf-8")) > max_size - 200:
            # Remove last line
            output = output[:output.rfind("\n")]
        output += "\n\n---\n_Output truncated to stay within size limit._\n"

    return output


# ── CLI ────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Generate a structured codebase summary for agent context."
    )
    sub = parser.add_subparsers(dest="command")

    gen = sub.add_parser("generate", help="Generate the project memory document")
    gen.add_argument(
        "--output", "-o",
        help="Output file path (default: stdout)",
        default=None,
    )
    gen.add_argument(
        "--max-depth",
        type=int,
        default=DEFAULT_MAX_DEPTH,
        help=f"Directory tree depth limit (default: {DEFAULT_MAX_DEPTH})",
    )
    gen.add_argument(
        "--max-size",
        type=int,
        default=DEFAULT_MAX_SIZE,
        help=f"Maximum output size in bytes (default: {DEFAULT_MAX_SIZE})",
    )
    gen.add_argument(
        "--root",
        default=".",
        help="Project root directory (default: current directory)",
    )

    args = parser.parse_args()

    if args.command is None:
        parser.print_help()
        sys.exit(1)

    if args.command == "generate":
        root = Path(args.root).resolve()
        if not root.is_dir():
            print(f"Error: {root} is not a directory", file=sys.stderr)
            sys.exit(1)

        output = generate_memory(root, args.max_depth, args.max_size)

        if args.output:
            out_path = Path(args.output)
            out_path.parent.mkdir(parents=True, exist_ok=True)
            out_path.write_text(output, encoding="utf-8")
            print(f"Memory written to {out_path} ({len(output.encode('utf-8')):,} bytes)", file=sys.stderr)
        else:
            print(output)


if __name__ == "__main__":
    main()
