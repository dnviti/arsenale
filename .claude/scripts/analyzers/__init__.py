"""Shared utilities for codebase analysis.

Provides language-agnostic file walking, language detection, pattern matching,
and file role classification. Zero external dependencies — stdlib only.
"""

import fnmatch
import json
import os
import re
from pathlib import Path
from typing import Generator

# ── Constants ───────────────────────────────────────────────────────────────

ALWAYS_SKIP = {
    ".git", "__pycache__", "node_modules", ".venv", "venv",
    ".tox", ".mypy_cache", ".pytest_cache", ".ruff_cache",
    "dist", "build", ".next", ".nuxt", "target",
    ".DS_Store", "Thumbs.db", ".eggs", "*.egg-info",
    ".cargo", ".gradle", ".m2", "vendor",
    "coverage", ".nyc_output", ".cache",
}

MAX_FILE_SIZE = 512_000  # 512KB per file

# Extension → (language, ecosystem)
EXTENSION_MAP: dict[str, tuple[str, str]] = {
    ".ts": ("TypeScript", "JavaScript"),
    ".tsx": ("TypeScript/React", "JavaScript"),
    ".js": ("JavaScript", "JavaScript"),
    ".jsx": ("JavaScript/React", "JavaScript"),
    ".mjs": ("JavaScript", "JavaScript"),
    ".cjs": ("JavaScript", "JavaScript"),
    ".py": ("Python", "Python"),
    ".pyi": ("Python", "Python"),
    ".rs": ("Rust", "Rust"),
    ".go": ("Go", "Go"),
    ".java": ("Java", "JVM"),
    ".kt": ("Kotlin", "JVM"),
    ".kts": ("Kotlin", "JVM"),
    ".scala": ("Scala", "JVM"),
    ".rb": ("Ruby", "Ruby"),
    ".erb": ("Ruby/ERB", "Ruby"),
    ".cs": ("C#", ".NET"),
    ".fs": ("F#", ".NET"),
    ".vb": ("VB.NET", ".NET"),
    ".php": ("PHP", "PHP"),
    ".swift": ("Swift", "Apple"),
    ".m": ("Objective-C", "Apple"),
    ".dart": ("Dart", "Dart"),
    ".lua": ("Lua", "Lua"),
    ".ex": ("Elixir", "BEAM"),
    ".exs": ("Elixir", "BEAM"),
    ".erl": ("Erlang", "BEAM"),
    ".zig": ("Zig", "Zig"),
    ".c": ("C", "C/C++"),
    ".cpp": ("C++", "C/C++"),
    ".cc": ("C++", "C/C++"),
    ".h": ("C/C++ Header", "C/C++"),
    ".hpp": ("C++ Header", "C/C++"),
    ".vue": ("Vue", "JavaScript"),
    ".svelte": ("Svelte", "JavaScript"),
    ".html": ("HTML", "Web"),
    ".css": ("CSS", "Web"),
    ".scss": ("SCSS", "Web"),
    ".less": ("LESS", "Web"),
    ".sql": ("SQL", "Database"),
    ".prisma": ("Prisma", "Database"),
    ".graphql": ("GraphQL", "API"),
    ".gql": ("GraphQL", "API"),
    ".proto": ("Protobuf", "API"),
    ".sh": ("Shell", "Scripts"),
    ".bash": ("Shell", "Scripts"),
    ".zsh": ("Shell", "Scripts"),
    ".ps1": ("PowerShell", "Scripts"),
}

# Framework detection: (file_pattern, framework_name)
FRAMEWORK_INDICATORS: list[tuple[str, str]] = [
    ("next.config.*", "Next.js"),
    ("nuxt.config.*", "Nuxt.js"),
    ("angular.json", "Angular"),
    ("vite.config.*", "Vite"),
    ("webpack.config.*", "Webpack"),
    ("svelte.config.*", "SvelteKit"),
    ("remix.config.*", "Remix"),
    ("astro.config.*", "Astro"),
    ("gatsby-config.*", "Gatsby"),
    ("manage.py", "Django"),
    ("app.py", "Flask/FastAPI"),
    ("fastapi", "FastAPI"),
    ("Cargo.toml", "Rust/Cargo"),
    ("go.mod", "Go Modules"),
    ("Gemfile", "Ruby/Bundler"),
    ("mix.exs", "Elixir/Mix"),
    ("pubspec.yaml", "Flutter/Dart"),
    ("composer.json", "PHP/Composer"),
    ("pom.xml", "Maven"),
    ("build.gradle", "Gradle"),
    ("build.gradle.kts", "Gradle (Kotlin)"),
    ("CMakeLists.txt", "CMake"),
    ("Makefile", "Make"),
    ("Dockerfile", "Docker"),
    ("docker-compose.yml", "Docker Compose"),
    ("compose.yml", "Docker Compose"),
]

# File role classification patterns
ROLE_PATTERNS: list[tuple[str, str]] = [
    # Routes / Controllers / Handlers
    (r"routes?[/\\]", "route"),
    (r"\.routes?\.", "route"),
    (r"controllers?[/\\]", "controller"),
    (r"\.controller\.", "controller"),
    (r"handlers?[/\\]", "handler"),
    (r"\.handler\.", "handler"),
    # Services / Business logic
    (r"services?[/\\]", "service"),
    (r"\.service\.", "service"),
    # Models / Schemas / Database
    (r"models?[/\\]", "model"),
    (r"\.model\.", "model"),
    (r"schemas?[/\\]", "schema"),
    (r"\.schema\.", "schema"),
    (r"migrations?[/\\]", "migration"),
    (r"prisma[/\\]", "schema"),
    # Components / Views / Templates
    (r"components?[/\\]", "component"),
    (r"views?[/\\]", "view"),
    (r"pages?[/\\]", "page"),
    (r"templates?[/\\]", "template"),
    (r"layouts?[/\\]", "layout"),
    # Middleware
    (r"middleware[/\\]", "middleware"),
    (r"\.middleware\.", "middleware"),
    # Tests
    (r"__tests__[/\\]", "test"),
    (r"tests?[/\\]", "test"),
    (r"\.test\.", "test"),
    (r"\.spec\.", "test"),
    (r"test_", "test"),
    (r"_test\.", "test"),
    # Configuration
    (r"\.config\.", "config"),
    (r"config[/\\]", "config"),
    # Utilities / Helpers
    (r"utils?[/\\]", "utility"),
    (r"helpers?[/\\]", "utility"),
    (r"lib[/\\]", "utility"),
    # Store / State
    (r"stores?[/\\]", "store"),
    (r"Store\.", "store"),
    # Hooks
    (r"hooks?[/\\]", "hook"),
    (r"use[A-Z]", "hook"),
    # API client
    (r"api[/\\]", "api-client"),
    (r"\.api\.", "api-client"),
    # CI/CD
    (r"\.github[/\\]workflows[/\\]", "ci-cd"),
    (r"\.gitlab-ci", "ci-cd"),
    (r"Jenkinsfile", "ci-cd"),
    # Docker / Container
    (r"Dockerfile", "container"),
    (r"compose.*\.yml", "container"),
    (r"\.dockerignore", "container"),
    # Documentation
    (r"docs?[/\\]", "documentation"),
    (r"\.md$", "documentation"),
]


# ── File System Utilities ───────────────────────────────────────────────────

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
        if fnmatch.fnmatch(name, pattern) or fnmatch.fnmatch(rel_path, pattern):
            return True
        if pattern.endswith("/"):
            dir_pattern = pattern.rstrip("/")
            if fnmatch.fnmatch(name, dir_pattern) or fnmatch.fnmatch(rel_path, dir_pattern):
                return True
    return False


def walk_source_files(
    root: Path,
    gitignore_patterns: list[str] | None = None,
    max_files: int = 5000,
) -> Generator[tuple[str, Path, str, int], None, None]:
    """Walk source files, yielding (rel_path, abs_path, extension, file_size).

    Skips ignored directories, binary files, and respects max_files limit.
    """
    if gitignore_patterns is None:
        gitignore_patterns = load_gitignore_patterns(root)

    count = 0
    for dirpath, dirnames, filenames in os.walk(root):
        # Prune ignored directories
        dirnames[:] = [
            d for d in dirnames
            if d not in ALWAYS_SKIP
            and not is_ignored(str(Path(dirpath, d).relative_to(root)), gitignore_patterns)
        ]
        dirnames.sort()

        for fname in sorted(filenames):
            fpath = Path(dirpath) / fname
            rel = str(fpath.relative_to(root))
            if is_ignored(rel, gitignore_patterns):
                continue
            try:
                size = fpath.stat().st_size
            except OSError:
                continue
            if size > MAX_FILE_SIZE:
                continue

            ext = fpath.suffix.lower()
            count += 1
            if count > max_files:
                return
            yield rel, fpath, ext, size


def read_file_safe(path: Path, max_size: int = MAX_FILE_SIZE) -> str:
    """Read file contents with encoding fallback and size limit."""
    try:
        if path.stat().st_size > max_size:
            return ""
        return path.read_text(encoding="utf-8", errors="replace")
    except (OSError, UnicodeDecodeError):
        return ""


# ── Language & Framework Detection ──────────────────────────────────────────

def detect_languages(root: Path, gitignore_patterns: list[str] | None = None) -> dict[str, int]:
    """Count source files by language. Returns {language: count} sorted by count."""
    counts: dict[str, int] = {}
    for _, _, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext in EXTENSION_MAP:
            lang = EXTENSION_MAP[ext][0]
            counts[lang] = counts.get(lang, 0) + 1
    return dict(sorted(counts.items(), key=lambda x: -x[1]))


def detect_ecosystems(root: Path, gitignore_patterns: list[str] | None = None) -> dict[str, int]:
    """Count source files by ecosystem. Returns {ecosystem: count}."""
    counts: dict[str, int] = {}
    for _, _, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext in EXTENSION_MAP:
            eco = EXTENSION_MAP[ext][1]
            counts[eco] = counts.get(eco, 0) + 1
    return dict(sorted(counts.items(), key=lambda x: -x[1]))


def detect_frameworks(root: Path) -> list[str]:
    """Detect frameworks by checking for indicator files."""
    found = []
    for pattern, framework in FRAMEWORK_INDICATORS:
        if "*" in pattern:
            if list(root.glob(pattern)):
                found.append(framework)
        else:
            if (root / pattern).exists():
                found.append(framework)
    return list(dict.fromkeys(found))  # deduplicate preserving order


# ── Pattern Matching ────────────────────────────────────────────────────────

def search_content(
    root: Path,
    pattern: str,
    gitignore_patterns: list[str] | None = None,
    extensions: set[str] | None = None,
    max_files: int = 5000,
    max_matches: int = 500,
) -> list[tuple[str, int, str]]:
    """Search file contents for a regex pattern.

    Returns list of (rel_path, line_number, matched_line).
    """
    regex = re.compile(pattern, re.IGNORECASE)
    results = []
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns, max_files):
        if extensions and ext not in extensions:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue
        for i, line in enumerate(content.splitlines(), 1):
            if regex.search(line):
                results.append((rel, i, line.strip()))
                if len(results) >= max_matches:
                    return results
    return results


def count_pattern(
    root: Path,
    pattern: str,
    gitignore_patterns: list[str] | None = None,
    extensions: set[str] | None = None,
    max_files: int = 5000,
) -> tuple[int, int]:
    """Count total matches and files matching a regex pattern.

    Returns (total_matches, files_with_matches).
    """
    regex = re.compile(pattern, re.IGNORECASE)
    total = 0
    files_hit = 0
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns, max_files):
        if extensions and ext not in extensions:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue
        matches = len(regex.findall(content))
        if matches:
            total += matches
            files_hit += 1
    return total, files_hit


# ── File Role Classification ────────────────────────────────────────────────

def classify_file_role(rel_path: str) -> str:
    """Classify a file's role based on its path and naming patterns."""
    normalized = rel_path.replace("\\", "/")
    for pattern, role in ROLE_PATTERNS:
        if re.search(pattern, normalized, re.IGNORECASE):
            return role
    return "other"


def classify_all_files(
    root: Path,
    gitignore_patterns: list[str] | None = None,
) -> dict[str, list[str]]:
    """Classify all files by role. Returns {role: [rel_paths]}."""
    roles: dict[str, list[str]] = {}
    for rel, _, _, _ in walk_source_files(root, gitignore_patterns):
        role = classify_file_role(rel)
        roles.setdefault(role, []).append(rel)
    return roles


# ── Manifest Parsing ────────────────────────────────────────────────────────

def parse_package_json(path: Path) -> dict:
    """Parse package.json and extract useful information."""
    if not path.exists():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8", errors="replace"))
    except (json.JSONDecodeError, OSError):
        return {}
    return {
        "name": data.get("name", ""),
        "version": data.get("version", ""),
        "scripts": data.get("scripts", {}),
        "dependencies": list(data.get("dependencies", {}).keys()),
        "devDependencies": list(data.get("devDependencies", {}).keys()),
        "workspaces": data.get("workspaces", []),
    }


def find_package_jsons(root: Path, gitignore_patterns: list[str] | None = None) -> list[Path]:
    """Find all package.json files in the project."""
    if gitignore_patterns is None:
        gitignore_patterns = load_gitignore_patterns(root)
    results = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [
            d for d in dirnames
            if d not in ALWAYS_SKIP
            and not is_ignored(str(Path(dirpath, d).relative_to(root)), gitignore_patterns)
        ]
        if "package.json" in filenames:
            results.append(Path(dirpath) / "package.json")
    return results


# ── Report Helpers ──────────────────────────────────────────────────────────

def make_table(headers: list[str], rows: list[list[str]]) -> str:
    """Generate a Markdown table."""
    if not rows:
        return "(none)\n"
    lines = []
    lines.append("| " + " | ".join(headers) + " |")
    lines.append("| " + " | ".join("---" for _ in headers) + " |")
    for row in rows:
        lines.append("| " + " | ".join(str(c) for c in row) + " |")
    return "\n".join(lines) + "\n"
