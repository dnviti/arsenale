"""Code Quality, Performance & Maintainability analyzer.

Detects test coverage ratios, complexity hotspots, code duplication,
error handling patterns, type safety issues, documentation quality,
naming conventions, and security anti-patterns.
Language-agnostic, zero external dependencies.
"""

import hashlib
import re
from collections import Counter
from pathlib import Path

from . import (
    classify_file_role,
    load_gitignore_patterns,
    make_table,
    read_file_safe,
    walk_source_files,
)

# Source file extensions (for analysis)
SOURCE_EXTS = {
    ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
    ".py", ".pyi",
    ".rs",
    ".go",
    ".java", ".kt", ".kts", ".scala",
    ".rb",
    ".cs", ".fs",
    ".php",
    ".swift",
    ".dart",
    ".ex", ".exs",
    ".c", ".cpp", ".cc", ".h", ".hpp",
    ".vue", ".svelte",
}

# ── Test Coverage ───────────────────────────────────────────────────────────

TEST_FILE_PATTERNS = [
    r"\.test\.", r"\.spec\.", r"_test\.", r"test_",
    r"__tests__[/\\]", r"tests?[/\\]",
]


def analyze_test_coverage(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze test file coverage by directory."""
    test_files: list[str] = []
    source_files: list[str] = []
    by_dir: dict[str, dict[str, int]] = {}  # dir -> {test: N, source: N}

    test_re = re.compile("|".join(TEST_FILE_PATTERNS), re.IGNORECASE)

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue

        # Determine directory (first 2 levels)
        parts = Path(rel).parts
        dir_key = str(Path(*parts[:2])) if len(parts) > 1 else parts[0] if parts else "."

        by_dir.setdefault(dir_key, {"test": 0, "source": 0})

        if test_re.search(rel):
            test_files.append(rel)
            by_dir[dir_key]["test"] += 1
        else:
            source_files.append(rel)
            by_dir[dir_key]["source"] += 1

    ratio = len(test_files) / max(len(source_files), 1) * 100

    # Test framework detection
    frameworks = set()
    pkg = root / "package.json"
    if pkg.exists():
        content = read_file_safe(pkg).lower()
        for fw in ["vitest", "jest", "mocha", "ava", "tap", "cypress", "playwright"]:
            if fw in content:
                frameworks.add(fw.capitalize())
    for fname in ["pytest.ini", "setup.cfg", "pyproject.toml", "conftest.py"]:
        if (root / fname).exists():
            content = read_file_safe(root / fname)
            if "pytest" in content.lower():
                frameworks.add("pytest")
    if (root / "phpunit.xml").exists() or (root / "phpunit.xml.dist").exists():
        frameworks.add("PHPUnit")

    return {
        "test_files": len(test_files),
        "source_files": len(source_files),
        "ratio_pct": round(ratio, 1),
        "by_directory": by_dir,
        "frameworks": sorted(frameworks),
    }


# ── Complexity Hotspots ─────────────────────────────────────────────────────

FUNCTION_PATTERNS = [
    r"^\s*(?:export\s+)?(?:async\s+)?function\s+\w+",           # JS/TS function
    r"^\s*(?:export\s+)?(?:const|let|var)\s+\w+\s*=\s*(?:async\s+)?\(", # arrow function
    r"^\s*(?:public|private|protected|static|async)*\s*\w+\s*\(.*\)\s*[:{]", # class method
    r"^\s*def\s+\w+\s*\(",                                       # Python
    r"^\s*fn\s+\w+",                                              # Rust
    r"^\s*func\s+(?:\(.*\)\s*)?\w+",                              # Go
    r"^\s*(?:public|private|protected|static)*\s*\w+\s+\w+\s*\(", # Java/C#
]


def analyze_complexity(root: Path, gitignore_patterns: list[str]) -> list[dict]:
    """Find complexity hotspots (large files, many functions, deep nesting)."""
    hotspots = []
    func_re = re.compile("|".join(FUNCTION_PATTERNS), re.MULTILINE)

    for rel, fpath, ext, size in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        lines = content.splitlines()
        line_count = len(lines)
        functions = len(func_re.findall(content))

        # Estimate max nesting by indentation
        max_indent = 0
        for line in lines:
            stripped = line.lstrip()
            if stripped:
                indent = len(line) - len(stripped)
                # Normalize: 2 or 4 spaces = 1 level, tab = 1 level
                if "\t" in line[:indent]:
                    level = line[:indent].count("\t")
                else:
                    level = indent // 2  # assume 2-space indent as minimum
                max_indent = max(max_indent, level)

        if line_count > 300 or functions > 15 or max_indent > 8:
            hotspots.append({
                "file": rel,
                "lines": line_count,
                "functions": functions,
                "max_nesting": max_indent,
            })

    # Sort by lines descending
    hotspots.sort(key=lambda x: -x["lines"])
    return hotspots[:30]  # top 30


# ── Code Duplication ────────────────────────────────────────────────────────

WINDOW_SIZE = 6  # lines per window for hash comparison


def analyze_duplication(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect potential code duplication using rolling hash windows."""
    hashes: dict[str, list[str]] = {}  # hash -> [file1, file2, ...]
    files_analyzed = 0

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        lines = [l.strip() for l in content.splitlines() if l.strip()]
        if len(lines) < WINDOW_SIZE:
            continue

        files_analyzed += 1
        seen_in_file: set[str] = set()  # avoid counting same file multiple times per hash

        for i in range(len(lines) - WINDOW_SIZE + 1):
            window = "\n".join(lines[i:i + WINDOW_SIZE])
            # Skip trivial windows (imports, blank-ish)
            if all(l.startswith(("import ", "from ", "//", "#", "/*", "*", "*/", "using "))
                   for l in lines[i:i + WINDOW_SIZE]):
                continue
            h = hashlib.md5(window.encode()).hexdigest()[:12]
            if h not in seen_in_file:
                seen_in_file.add(h)
                hashes.setdefault(h, []).append(rel)

    # Find hashes appearing in multiple files
    duplicated_blocks = 0
    file_pairs: Counter = Counter()
    for h, files in hashes.items():
        unique_files = list(set(files))
        if len(unique_files) > 1:
            duplicated_blocks += 1
            for i in range(len(unique_files)):
                for j in range(i + 1, len(unique_files)):
                    pair = tuple(sorted([unique_files[i], unique_files[j]]))
                    file_pairs[pair] += 1

    return {
        "files_analyzed": files_analyzed,
        "duplicated_blocks": duplicated_blocks,
        "top_pairs": file_pairs.most_common(10),
    }


# ── Error Handling ──────────────────────────────────────────────────────────

def analyze_error_handling(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze error handling patterns."""
    try_catch_count = 0
    empty_catch_count = 0
    unhandled_promise_count = 0
    files_with_issues = 0

    try_catch_re = re.compile(r'\btry\s*\{', re.MULTILINE)
    # Empty catch: catch(...) { } or except ...: pass
    empty_catch_re = re.compile(
        r'catch\s*\([^)]*\)\s*\{\s*\}|except\s+\w+.*:\s*pass\s*$',
        re.MULTILINE
    )
    unhandled_re = re.compile(r'\.then\s*\([^)]*\)\s*(?!\s*\.catch)', re.MULTILINE)

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        tc = len(try_catch_re.findall(content))
        ec = len(empty_catch_re.findall(content))
        up = len(unhandled_re.findall(content))

        try_catch_count += tc
        empty_catch_count += ec
        unhandled_promise_count += up

        if ec > 0 or up > 0:
            files_with_issues += 1

    return {
        "try_catch_blocks": try_catch_count,
        "empty_catch_blocks": empty_catch_count,
        "unhandled_promises": unhandled_promise_count,
        "files_with_issues": files_with_issues,
    }


# ── Type Safety ─────────────────────────────────────────────────────────────

def analyze_type_safety(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze type safety issues (TypeScript any, ignores, etc.)."""
    any_count = 0
    any_files = 0
    ts_ignore_count = 0
    eslint_disable_count = 0
    type_ignore_count = 0  # Python type: ignore

    ts_exts = {".ts", ".tsx"}
    py_exts = {".py"}

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext in ts_exts:
            content = read_file_safe(fpath)
            if not content:
                continue
            # Count `: any` and `as any` but not in comments
            any_matches = len(re.findall(r'(?::\s*any\b|as\s+any\b)', content))
            if any_matches:
                any_count += any_matches
                any_files += 1
            ts_ignore_count += len(re.findall(r'@ts-ignore|@ts-expect-error', content))
            eslint_disable_count += len(re.findall(r'eslint-disable', content))
        elif ext in py_exts:
            content = read_file_safe(fpath)
            if content:
                type_ignore_count += len(re.findall(r'type:\s*ignore', content))

    return {
        "any_usage": any_count,
        "any_files": any_files,
        "ts_ignore": ts_ignore_count,
        "eslint_disable": eslint_disable_count,
        "type_ignore_python": type_ignore_count,
    }


# ── Documentation Quality ──────────────────────────────────────────────────

def analyze_documentation(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze documentation quality."""
    doc_comment_patterns = [
        r'/\*\*',           # JSDoc / JavaDoc
        r'"""',              # Python docstring
        r"'''",              # Python docstring (alt)
        r'///\s',            # Rust / C# doc comments
        r'///',              # Go godoc (function-level)
    ]
    doc_re = re.compile("|".join(doc_comment_patterns))

    files_with_docs = 0
    files_without_docs = 0
    total_doc_comments = 0

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        role = classify_file_role(rel)
        # Only check service/controller/route/utility files
        if role not in ("service", "controller", "route", "handler", "utility", "hook", "other"):
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        doc_count = len(doc_re.findall(content))
        total_doc_comments += doc_count
        if doc_count > 0:
            files_with_docs += 1
        else:
            files_without_docs += 1

    # Documentation files
    docs_dir = root / "docs"
    doc_files = 0
    if docs_dir.is_dir():
        doc_files = len([f for f in docs_dir.rglob("*.md")])

    readme_exists = (root / "README.md").exists()
    readme_lines = 0
    if readme_exists:
        content = read_file_safe(root / "README.md")
        readme_lines = len(content.splitlines())

    return {
        "files_with_doc_comments": files_with_docs,
        "files_without_doc_comments": files_without_docs,
        "total_doc_comments": total_doc_comments,
        "doc_files_in_docs": doc_files,
        "readme_exists": readme_exists,
        "readme_lines": readme_lines,
    }


# ── Naming Conventions ──────────────────────────────────────────────────────

def analyze_naming(root: Path, gitignore_patterns: list[str]) -> dict:
    """Check file naming consistency."""
    conventions: Counter = Counter()  # pattern -> count

    for rel, _, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        name = Path(rel).stem
        if not name:
            continue

        if "-" in name and "_" not in name and name == name.lower():
            conventions["kebab-case"] += 1
        elif "_" in name and "-" not in name and name == name.lower():
            conventions["snake_case"] += 1
        elif name[0].isupper() and "_" not in name and "-" not in name:
            conventions["PascalCase"] += 1
        elif name[0].islower() and "_" not in name and "-" not in name:
            conventions["camelCase"] += 1
        else:
            conventions["mixed"] += 1

    dominant = conventions.most_common(1)[0] if conventions else ("unknown", 0)
    inconsistent = sum(c for p, c in conventions.items() if p != dominant[0])

    return {
        "conventions": dict(conventions.most_common()),
        "dominant": dominant[0],
        "inconsistent_files": inconsistent,
    }


# ── Security Patterns ──────────────────────────────────────────────────────

SECRET_PATTERNS = [
    (r"(?:password|passwd|pwd)\s*=\s*['\"][^'\"]{8,}['\"]", "Hardcoded password"),
    (r"(?:api[_-]?key|apikey)\s*=\s*['\"][^'\"]{16,}['\"]", "Hardcoded API key"),
    (r"(?:secret|token)\s*=\s*['\"][A-Za-z0-9+/=]{20,}['\"]", "Hardcoded secret/token"),
    (r"-----BEGIN (?:RSA )?PRIVATE KEY-----", "Private key in source"),
]

SQL_INJECTION_PATTERNS = [
    r"(?:query|execute)\s*\(\s*['\"].*\$\{",          # Template literal in SQL
    r"(?:query|execute)\s*\(\s*['\"].*\+\s*\w+",       # String concatenation in SQL
    r"(?:query|execute)\s*\(\s*f['\"]",                 # Python f-string in SQL
    r"(?:query|execute)\s*\(\s*['\"].*%s.*%\s*\(",      # Python % formatting
]


def analyze_security(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze security patterns and anti-patterns."""
    issues: list[dict] = []

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        role = classify_file_role(rel)
        if role in ("test", "documentation"):
            continue  # skip tests and docs
        content = read_file_safe(fpath)
        if not content:
            continue

        for pattern, label in SECRET_PATTERNS:
            if re.search(pattern, content, re.IGNORECASE):
                issues.append({"file": rel, "type": label})

        for pattern in SQL_INJECTION_PATTERNS:
            if re.search(pattern, content, re.IGNORECASE):
                issues.append({"file": rel, "type": "Potential SQL injection"})

    # Check for security middleware/headers
    has_helmet = False
    has_cors_config = False
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        content = read_file_safe(fpath)
        if content:
            if re.search(r"helmet\b", content, re.IGNORECASE):
                has_helmet = True
            if re.search(r"cors\b.*origin", content, re.IGNORECASE):
                has_cors_config = True

    return {
        "issues": issues[:20],  # limit output
        "has_security_headers": has_helmet,
        "has_cors_config": has_cors_config,
    }


# ── Report Generator ────────────────────────────────────────────────────────

def generate_report(root: Path, gitignore_patterns: list[str] | None = None) -> str:
    """Generate the full code quality report."""
    root = root.resolve()
    if gitignore_patterns is None:
        gitignore_patterns = load_gitignore_patterns(root)

    sections = []
    sections.append("# Code Quality, Performance & Maintainability Report\n")
    sections.append(f"> Auto-generated static analysis for `{root.name}`\n")

    # Test Coverage
    tests = analyze_test_coverage(root, gitignore_patterns)
    sections.append("## Test Coverage\n")
    sections.append(f"- **Test files:** {tests['test_files']}")
    sections.append(f"- **Source files:** {tests['source_files']}")
    sections.append(f"- **Test-to-source ratio:** {tests['ratio_pct']}%")
    if tests["frameworks"]:
        sections.append(f"- **Test frameworks:** {', '.join(tests['frameworks'])}")
    sections.append("")

    if tests["by_directory"]:
        dir_rows = []
        for d, counts in sorted(tests["by_directory"].items()):
            if counts["source"] > 0 or counts["test"] > 0:
                ratio = round(counts["test"] / max(counts["source"], 1) * 100, 1)
                dir_rows.append([d, str(counts["source"]), str(counts["test"]), f"{ratio}%"])
        if dir_rows:
            sections.append(make_table(
                ["Directory", "Source", "Tests", "Ratio"],
                dir_rows[:20],
            ))

    # Complexity Hotspots
    hotspots = analyze_complexity(root, gitignore_patterns)
    sections.append("## Complexity Hotspots\n")
    if hotspots:
        sections.append(make_table(
            ["File", "Lines", "Functions", "Max Nesting"],
            [[h["file"], str(h["lines"]), str(h["functions"]), str(h["max_nesting"])]
             for h in hotspots[:15]],
        ))
    else:
        sections.append("No significant complexity hotspots detected.\n")

    # Code Duplication
    duplication = analyze_duplication(root, gitignore_patterns)
    sections.append("## Code Duplication\n")
    sections.append(f"- **Files analyzed:** {duplication['files_analyzed']}")
    sections.append(f"- **Duplicated code blocks:** {duplication['duplicated_blocks']}")
    if duplication["top_pairs"]:
        sections.append("\n**Top file pairs with shared code:**\n")
        for (f1, f2), count in duplication["top_pairs"][:10]:
            sections.append(f"- `{f1}` ↔ `{f2}` ({count} shared blocks)")
    sections.append("")

    # Error Handling
    errors = analyze_error_handling(root, gitignore_patterns)
    sections.append("## Error Handling\n")
    sections.append(f"- **try/catch blocks:** {errors['try_catch_blocks']}")
    sections.append(f"- **Empty catch blocks:** {errors['empty_catch_blocks']}" +
                   (" ⚠️" if errors['empty_catch_blocks'] > 0 else ""))
    sections.append(f"- **Unhandled promise patterns:** {errors['unhandled_promises']}" +
                   (" ⚠️" if errors['unhandled_promises'] > 0 else ""))
    sections.append(f"- **Files with issues:** {errors['files_with_issues']}")
    sections.append("")

    # Type Safety
    types = analyze_type_safety(root, gitignore_patterns)
    sections.append("## Type Safety\n")
    if types["any_usage"] or types["ts_ignore"] or types["eslint_disable"]:
        sections.append(f"- **`any` usage:** {types['any_usage']} occurrences in {types['any_files']} files" +
                       (" ⚠️" if types['any_usage'] > 10 else ""))
        sections.append(f"- **@ts-ignore / @ts-expect-error:** {types['ts_ignore']}")
        sections.append(f"- **eslint-disable:** {types['eslint_disable']}")
    if types["type_ignore_python"]:
        sections.append(f"- **Python type: ignore:** {types['type_ignore_python']}")
    if not any(types.values()):
        sections.append("No type safety issues detected (or project doesn't use typed languages).")
    sections.append("")

    # Documentation Quality
    docs = analyze_documentation(root, gitignore_patterns)
    sections.append("## Documentation Quality\n")
    total_checked = docs["files_with_doc_comments"] + docs["files_without_doc_comments"]
    if total_checked:
        pct = round(docs["files_with_doc_comments"] / total_checked * 100, 1)
        sections.append(f"- **Files with doc comments:** {docs['files_with_doc_comments']}/{total_checked} ({pct}%)")
    sections.append(f"- **Total doc comments:** {docs['total_doc_comments']}")
    sections.append(f"- **docs/ directory:** {docs['doc_files_in_docs']} files" if docs['doc_files_in_docs'] else "- **docs/ directory:** not found")
    sections.append(f"- **README.md:** {'exists' if docs['readme_exists'] else 'not found'}" +
                   (f" ({docs['readme_lines']} lines)" if docs['readme_exists'] else ""))
    sections.append("")

    # Naming Conventions
    naming = analyze_naming(root, gitignore_patterns)
    sections.append("## Naming Conventions\n")
    if naming["conventions"]:
        sections.append(f"- **Dominant convention:** {naming['dominant']}")
        sections.append(f"- **Inconsistent files:** {naming['inconsistent_files']}")
        sections.append(make_table(
            ["Convention", "Files"],
            [[conv, str(count)] for conv, count in naming["conventions"].items()],
        ))

    # Security
    security = analyze_security(root, gitignore_patterns)
    sections.append("## Security Practices\n")
    sections.append(f"- **Security headers (Helmet):** {'detected' if security['has_security_headers'] else 'not detected'}")
    sections.append(f"- **CORS configuration:** {'detected' if security['has_cors_config'] else 'not detected'}")
    if security["issues"]:
        sections.append(f"\n**⚠️ Security issues found ({len(security['issues'])}):**\n")
        for issue in security["issues"]:
            sections.append(f"- `{issue['file']}`: {issue['type']}")
    else:
        sections.append("- **No hardcoded secrets or injection patterns detected**")
    sections.append("")

    # Technical Debt Summary
    sections.append("## Technical Debt Summary\n")
    debt_items = []
    if tests["ratio_pct"] < 10:
        debt_items.append(f"Low test coverage ({tests['ratio_pct']}% test-to-source ratio)")
    if errors["empty_catch_blocks"] > 0:
        debt_items.append(f"{errors['empty_catch_blocks']} empty catch blocks swallowing errors")
    if types["any_usage"] > 20:
        debt_items.append(f"Excessive `any` type usage ({types['any_usage']} occurrences)")
    if types["ts_ignore"] > 5:
        debt_items.append(f"{types['ts_ignore']} TypeScript ignore directives")
    if naming["inconsistent_files"] > 10:
        debt_items.append(f"Inconsistent file naming ({naming['inconsistent_files']} files deviate from {naming['dominant']})")
    if len(hotspots) > 5:
        debt_items.append(f"{len(hotspots)} complexity hotspots (files >300 lines or >15 functions)")
    if duplication["duplicated_blocks"] > 20:
        debt_items.append(f"{duplication['duplicated_blocks']} potential code duplication blocks")
    if not docs["readme_exists"]:
        debt_items.append("Missing README.md")

    if debt_items:
        for item in debt_items:
            sections.append(f"- {item}")
    else:
        sections.append("No significant technical debt detected.")
    sections.append("")

    return "\n".join(sections)
