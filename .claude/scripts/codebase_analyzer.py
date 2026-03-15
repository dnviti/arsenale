#!/usr/bin/env python3
"""Codebase analyzer for the agentic fleet pipeline.

Generates structured Markdown reports on infrastructure, features, and
code quality — replacing expensive LLM-based reader agents with
deterministic static analysis.

Zero external dependencies — stdlib only.

Usage:
    # All three reports
    python3 codebase_analyzer.py analyze --output-dir .

    # Single focus area
    python3 codebase_analyzer.py analyze --focus infrastructure --output report.md

    # With memory file and custom root
    python3 codebase_analyzer.py analyze --root /path/to/project --output-dir . --memory project-memory.md
"""

import argparse
import sys
import time
from pathlib import Path

# Add parent directory to path so analyzers package can be found
# when running from any location
_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))

from analyzers import load_gitignore_patterns
from analyzers.infrastructure import generate_report as gen_infrastructure
from analyzers.features import generate_report as gen_features
from analyzers.quality import generate_report as gen_quality

FOCUS_AREAS = {
    "infrastructure": ("report-infrastructure.md", gen_infrastructure),
    "features": ("report-features.md", gen_features),
    "quality": ("report-quality.md", gen_quality),
}


def analyze(root: Path, focus: list[str], output_dir: Path | None, output: Path | None):
    """Run analysis for the specified focus areas."""
    root = root.resolve()
    if not root.is_dir():
        print(f"Error: {root} is not a directory", file=sys.stderr)
        sys.exit(1)

    gitignore_patterns = load_gitignore_patterns(root)

    for area in focus:
        if area not in FOCUS_AREAS:
            print(f"Error: unknown focus area '{area}'. Valid: {', '.join(FOCUS_AREAS)}", file=sys.stderr)
            sys.exit(1)

        default_filename, generator = FOCUS_AREAS[area]
        start = time.time()

        print(f"Analyzing: {area}...", file=sys.stderr, end=" ", flush=True)
        report = generator(root, gitignore_patterns)
        elapsed = time.time() - start
        print(f"done ({elapsed:.1f}s)", file=sys.stderr)

        # Determine output path
        if output and len(focus) == 1:
            out_path = output
        elif output_dir:
            out_path = output_dir / default_filename
        else:
            # stdout
            print(report)
            continue

        out_path.parent.mkdir(parents=True, exist_ok=True)
        out_path.write_text(report, encoding="utf-8")
        size = len(report.encode("utf-8"))
        print(f"  Written to {out_path} ({size:,} bytes)", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(
        description="Generate codebase analysis reports for agent context (replaces Sonnet reader agents)."
    )
    sub = parser.add_subparsers(dest="command")

    analyze_cmd = sub.add_parser(
        "analyze",
        help="Analyze the codebase and generate reports",
    )
    analyze_cmd.add_argument(
        "--root",
        default=".",
        help="Project root directory (default: current directory)",
    )
    analyze_cmd.add_argument(
        "--focus",
        default="infrastructure,features,quality",
        help="Comma-separated focus areas (default: all three)",
    )
    analyze_cmd.add_argument(
        "--output-dir",
        default=None,
        help="Directory to write report files (uses default filenames)",
    )
    analyze_cmd.add_argument(
        "--output", "-o",
        default=None,
        help="Output file path (only valid with a single focus area)",
    )
    analyze_cmd.add_argument(
        "--memory",
        default=None,
        help="Path to project-memory.md (currently unused, reserved for future)",
    )

    args = parser.parse_args()

    if args.command is None:
        parser.print_help()
        sys.exit(1)

    if args.command == "analyze":
        focus = [f.strip() for f in args.focus.split(",")]
        output_dir = Path(args.output_dir) if args.output_dir else None
        output = Path(args.output) if args.output else None

        if output and len(focus) > 1:
            print("Error: --output can only be used with a single --focus area", file=sys.stderr)
            sys.exit(1)

        analyze(Path(args.root), focus, output_dir, output)


if __name__ == "__main__":
    main()
