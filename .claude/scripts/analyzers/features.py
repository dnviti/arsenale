"""Features & User Experience analyzer.

Detects UI components, API endpoints, authentication mechanisms,
state management, real-time features, accessibility, i18n,
and feature gaps. Language-agnostic, zero external dependencies.
"""

import re
from collections import Counter
from pathlib import Path

from . import (
    classify_all_files,
    count_pattern,
    detect_languages,
    load_gitignore_patterns,
    make_table,
    read_file_safe,
    search_content,
    walk_source_files,
)

# Source file extensions
SOURCE_EXTS = {
    ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
    ".py", ".pyi", ".rs", ".go",
    ".java", ".kt", ".scala",
    ".rb", ".cs", ".php", ".swift", ".dart",
    ".vue", ".svelte",
    ".ex", ".exs",
}

UI_EXTS = {".tsx", ".jsx", ".vue", ".svelte", ".html"}

# ── UI Component Inventory ──────────────────────────────────────────────────

def analyze_ui_components(root: Path, gitignore_patterns: list[str]) -> dict:
    """Inventory UI components grouped by domain."""
    components: dict[str, list[dict]] = {}  # domain -> [{name, file, lines}]
    total_lines = 0
    total_components = 0

    for rel, fpath, ext, size in walk_source_files(root, gitignore_patterns):
        if ext not in UI_EXTS:
            continue

        # Classify domain from path
        parts = Path(rel).parts
        domain = "root"
        # Look for meaningful directory name
        for part in parts:
            if part.lower() in ("components", "views", "pages", "layouts", "src",
                                "app", "client", "frontend", "ui"):
                continue
            if part == Path(rel).name:
                continue
            domain = part
            break

        content = read_file_safe(fpath)
        lines = len(content.splitlines()) if content else 0
        total_lines += lines
        total_components += 1

        components.setdefault(domain, []).append({
            "name": Path(rel).stem,
            "file": rel,
            "lines": lines,
        })

    return {
        "by_domain": components,
        "total_components": total_components,
        "total_lines": total_lines,
    }


# ── API Endpoint Summary ───────────────────────────────────────────────────

API_ROUTE_PATTERNS = [
    # Express.js
    (r"(?:router|app)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['\"]([^'\"]+)['\"]", "Express"),
    # FastAPI / Flask decorators
    (r"@(?:app|router)\.(get|post|put|patch|delete|options|head)\s*\(\s*['\"]([^'\"]+)['\"]", "FastAPI/Flask"),
    # Spring annotations
    (r"@(Get|Post|Put|Patch|Delete)Mapping\s*\(\s*(?:value\s*=\s*)?['\"]([^'\"]+)['\"]", "Spring"),
    # Go
    (r"(?:Handle|HandleFunc)\s*\(\s*['\"]([^'\"]+)['\"]", "Go"),
    # Rails
    (r"^\s*(get|post|put|patch|delete)\s+['\"]([^'\"]+)['\"]", "Rails"),
    # ASP.NET
    (r"\[Http(Get|Post|Put|Patch|Delete)\s*\(\s*['\"]([^'\"]+)['\"]\s*\)\]", "ASP.NET"),
]


def analyze_api_endpoints(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze API endpoints with domain grouping."""
    endpoints: list[dict] = []
    framework = None

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        for pattern, fw_name in API_ROUTE_PATTERNS:
            for match in re.finditer(pattern, content, re.MULTILINE):
                groups = match.groups()
                if len(groups) == 2:
                    method, path = groups[0].upper(), groups[1]
                elif len(groups) == 1:
                    method, path = "ANY", groups[0]
                else:
                    continue
                endpoints.append({"method": method, "path": path, "file": rel})
                if not framework:
                    framework = fw_name

    # Group by domain
    by_domain: dict[str, dict[str, int]] = {}
    for ep in endpoints:
        parts = ep["path"].strip("/").split("/")
        domain = parts[0] if parts and parts[0] else "root"
        by_domain.setdefault(domain, Counter())
        by_domain[domain][ep["method"]] += 1

    return {
        "framework": framework,
        "total": len(endpoints),
        "by_domain": by_domain,
    }


# ── Authentication Mechanisms ───────────────────────────────────────────────

AUTH_INDICATORS = [
    ("JWT", [r"jsonwebtoken", r"jwt\.sign", r"jwt\.verify", r"JwtModule", r"jose\b"]),
    ("OAuth", [r"oauth2?\b", r"passport\b", r"openid", r"authorization_code"]),
    ("SAML", [r"saml\b", r"passport-saml"]),
    ("LDAP", [r"ldap\b", r"ldapjs", r"activedirectory"]),
    ("WebAuthn/Passkeys", [r"webauthn", r"passkey", r"fido2", r"@simplewebauthn"]),
    ("TOTP/2FA", [r"totp\b", r"speakeasy", r"otplib", r"two.?factor", r"2fa\b", r"mfa\b"]),
    ("SMS/Phone Auth", [r"twilio", r"sms.?auth", r"phone.?verif"]),
    ("API Keys", [r"api.?key", r"x-api-key", r"apikey"]),
    ("Session-based", [r"express-session", r"cookie-session", r"session.?store"]),
    ("Basic Auth", [r"basic.?auth", r"authorization.*basic"]),
]


def detect_auth_mechanisms(root: Path, gitignore_patterns: list[str]) -> list[str]:
    """Detect authentication mechanisms used in the project."""
    found = []
    for name, patterns in AUTH_INDICATORS:
        for pattern in patterns:
            matches, files = count_pattern(root, pattern, gitignore_patterns, SOURCE_EXTS)
            if matches > 0:
                found.append(name)
                break
    return found


# ── State Management ────────────────────────────────────────────────────────

STATE_MGMT_LIBS = [
    ("Zustand", [r"from\s+['\"]zustand['\"]", r"import.*zustand", r"create\s*\("]),
    ("Redux", [r"from\s+['\"](?:redux|@reduxjs)['\"]", r"createStore", r"configureStore", r"createSlice"]),
    ("MobX", [r"from\s+['\"]mobx['\"]", r"observable", r"makeAutoObservable"]),
    ("Pinia", [r"from\s+['\"]pinia['\"]", r"defineStore"]),
    ("Vuex", [r"from\s+['\"]vuex['\"]", r"createStore"]),
    ("Recoil", [r"from\s+['\"]recoil['\"]", r"atom\(", r"selector\("]),
    ("Jotai", [r"from\s+['\"]jotai['\"]"]),
    ("XState", [r"from\s+['\"]xstate['\"]", r"createMachine"]),
    ("NgRx", [r"from\s+['\"]@ngrx['\"]", r"createAction", r"createReducer"]),
    ("Context API", [r"createContext\s*\(", r"useContext\s*\("]),
]


def detect_state_management(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect state management libraries and list stores."""
    libraries = []
    stores: list[dict] = []

    for lib_name, patterns in STATE_MGMT_LIBS:
        for pattern in patterns:
            matches, files = count_pattern(root, pattern, gitignore_patterns, SOURCE_EXTS)
            if matches > 0:
                libraries.append(lib_name)
                break

    # Find store files
    store_re = re.compile(r"(?:Store|store|_store)\.", re.IGNORECASE)
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in SOURCE_EXTS:
            continue
        if store_re.search(rel):
            stores.append({"file": rel, "name": Path(rel).stem})

    return {"libraries": libraries, "stores": stores}


# ── Real-time Features ──────────────────────────────────────────────────────

REALTIME_INDICATORS = [
    ("WebSocket", [r"WebSocket\b", r"ws\b.*import", r"wss?://"]),
    ("Socket.IO", [r"socket\.io", r"io\s*\(\s*['\"]", r"socketio"]),
    ("Server-Sent Events", [r"EventSource\b", r"text/event-stream"]),
    ("GraphQL Subscriptions", [r"subscription\b.*\{", r"useSubscription"]),
    ("MQTT", [r"mqtt\b", r"MqttClient"]),
    ("gRPC Streaming", [r"grpc.*stream", r"ServerStream", r"BidiStream"]),
]


def detect_realtime(root: Path, gitignore_patterns: list[str]) -> list[str]:
    """Detect real-time communication patterns."""
    found = []
    for name, patterns in REALTIME_INDICATORS:
        for pattern in patterns:
            matches, files = count_pattern(root, pattern, gitignore_patterns, SOURCE_EXTS)
            if matches > 0:
                found.append(name)
                break
    return found


# ── Accessibility ───────────────────────────────────────────────────────────

def analyze_accessibility(root: Path, gitignore_patterns: list[str]) -> dict:
    """Analyze accessibility practices in UI code."""
    aria_count = 0
    role_count = 0
    tabindex_count = 0
    alt_count = 0
    a11y_files = 0
    total_ui_files = 0

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in UI_EXTS:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        total_ui_files += 1
        has_a11y = False

        a = len(re.findall(r'aria-\w+', content))
        r = len(re.findall(r'role\s*=', content))
        t = len(re.findall(r'tabIndex|tabindex', content))
        alt = len(re.findall(r'alt\s*=', content))

        aria_count += a
        role_count += r
        tabindex_count += t
        alt_count += alt

        if a + r + t + alt > 0:
            a11y_files += 1

    # Check for a11y libraries
    a11y_libs = []
    pkg = root / "package.json"
    if pkg.exists():
        content = read_file_safe(pkg).lower()
        for lib in ["@axe-core", "react-aria", "@radix-ui", "eslint-plugin-jsx-a11y",
                     "@testing-library", "pa11y", "lighthouse"]:
            if lib in content:
                a11y_libs.append(lib)

    return {
        "aria_attributes": aria_count,
        "role_attributes": role_count,
        "tabindex_usage": tabindex_count,
        "alt_attributes": alt_count,
        "files_with_a11y": a11y_files,
        "total_ui_files": total_ui_files,
        "a11y_libraries": a11y_libs,
    }


# ── Internationalization ───────────────────────────────────────────────────

I18N_INDICATORS = [
    ("react-intl", [r"react-intl", r"FormattedMessage", r"useIntl"]),
    ("i18next", [r"i18next", r"react-i18next", r"useTranslation", r"t\(['\"]"]),
    ("vue-i18n", [r"vue-i18n", r"\$t\("]),
    ("gettext", [r"gettext", r"ngettext", r"_\(['\"]"]),
    ("Fluent", [r"@fluent", r"fluent-react"]),
    ("FormatJS", [r"@formatjs"]),
    ("lingui", [r"@lingui"]),
]


def detect_i18n(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect internationalization setup."""
    libraries = []
    for name, patterns in I18N_INDICATORS:
        for pattern in patterns:
            matches, _ = count_pattern(root, pattern, gitignore_patterns, SOURCE_EXTS)
            if matches > 0:
                libraries.append(name)
                break

    # Check for locale files
    locale_dirs = []
    for d in ["locales", "locale", "i18n", "translations", "lang", "messages"]:
        path = root / d
        if path.is_dir():
            locale_dirs.append(d)
        # Also check in src/
        src_path = root / "src" / d
        if src_path.is_dir():
            locale_dirs.append(f"src/{d}")

    locale_files = []
    for d in locale_dirs:
        dp = root / d
        for f in dp.rglob("*"):
            if f.suffix in (".json", ".yaml", ".yml", ".po", ".mo", ".ftl", ".xlf"):
                locale_files.append(str(f.relative_to(root)))

    return {
        "libraries": libraries,
        "locale_dirs": locale_dirs,
        "locale_files": len(locale_files),
    }


# ── Feature Gap Detection ──────────────────────────────────────────────────

def detect_feature_gaps(
    root: Path,
    gitignore_patterns: list[str],
    auth: list[str],
    i18n: dict,
    a11y: dict,
    realtime: list[str],
    state: dict,
) -> list[str]:
    """Detect feature gaps based on absence patterns."""
    gaps = []

    if not i18n["libraries"] and not i18n["locale_dirs"]:
        gaps.append("No internationalization (i18n) infrastructure detected")

    if a11y["total_ui_files"] > 0:
        a11y_ratio = a11y["files_with_a11y"] / max(a11y["total_ui_files"], 1)
        if a11y_ratio < 0.3:
            gaps.append(f"Low accessibility coverage — only {a11y['files_with_a11y']}/{a11y['total_ui_files']} UI files have ARIA/role attributes")

    if not a11y["a11y_libraries"]:
        gaps.append("No dedicated accessibility testing library detected")

    # Check for onboarding/wizard
    onboarding, _ = count_pattern(root, r"onboarding|wizard|tour|welcome.*step|setup.*wizard",
                                   gitignore_patterns, SOURCE_EXTS)
    if onboarding == 0:
        gaps.append("No onboarding/wizard flow detected")

    # Check for search functionality
    search, _ = count_pattern(root, r"search.*component|SearchBar|search.*input|useSearch",
                               gitignore_patterns, UI_EXTS)
    if search == 0 and a11y["total_ui_files"] > 10:
        gaps.append("No search functionality detected in UI")

    # Check for error boundary
    error_boundary, _ = count_pattern(root, r"ErrorBoundary|error.?boundary|componentDidCatch",
                                       gitignore_patterns, UI_EXTS)
    if error_boundary == 0 and a11y["total_ui_files"] > 5:
        gaps.append("No error boundary component detected")

    # Check for loading/skeleton states
    skeleton, _ = count_pattern(root, r"Skeleton|skeleton|shimmer|placeholder.*loading",
                                 gitignore_patterns, UI_EXTS)
    if skeleton == 0 and a11y["total_ui_files"] > 10:
        gaps.append("No skeleton/shimmer loading states detected")

    # Check for dark mode
    dark_mode, _ = count_pattern(root, r"dark.?mode|theme.?toggle|prefers-color-scheme|darkMode|ThemeProvider",
                                  gitignore_patterns)
    if dark_mode == 0 and a11y["total_ui_files"] > 5:
        gaps.append("No dark mode / theme toggle detected")

    # Check for notification/toast system
    notifications, _ = count_pattern(root, r"toast|snackbar|notification.*provider|useNotif",
                                      gitignore_patterns, SOURCE_EXTS)
    if notifications == 0 and a11y["total_ui_files"] > 5:
        gaps.append("No toast/notification system detected")

    return gaps


# ── Report Generator ────────────────────────────────────────────────────────

def generate_report(root: Path, gitignore_patterns: list[str] | None = None) -> str:
    """Generate the full features & user experience report."""
    root = root.resolve()
    if gitignore_patterns is None:
        gitignore_patterns = load_gitignore_patterns(root)

    sections = []
    sections.append("# Features & User Experience Report\n")
    sections.append(f"> Auto-generated static analysis for `{root.name}`\n")

    # UI Components
    ui = analyze_ui_components(root, gitignore_patterns)
    if ui["total_components"] > 0:
        sections.append("## UI Component Inventory\n")
        sections.append(f"- **Total components:** {ui['total_components']}")
        sections.append(f"- **Total lines:** {ui['total_lines']:,}\n")

        domain_rows = []
        for domain, comps in sorted(ui["by_domain"].items()):
            total_lines = sum(c["lines"] for c in comps)
            domain_rows.append([domain, str(len(comps)), f"{total_lines:,}"])
        sections.append(make_table(
            ["Domain", "Components", "Lines"],
            domain_rows[:20],
        ))

    # API Endpoints
    api = analyze_api_endpoints(root, gitignore_patterns)
    if api["total"] > 0:
        sections.append("## API Endpoint Summary\n")
        sections.append(f"- **Framework:** {api['framework'] or 'Unknown'}")
        sections.append(f"- **Total endpoints:** {api['total']}\n")

        domain_rows = []
        for domain, methods in sorted(api["by_domain"].items()):
            domain_rows.append([
                domain,
                str(methods.get("GET", 0)),
                str(methods.get("POST", 0)),
                str(methods.get("PUT", 0)),
                str(methods.get("DELETE", 0)),
                str(sum(methods.values())),
            ])
        sections.append(make_table(
            ["Domain", "GET", "POST", "PUT", "DELETE", "Total"],
            domain_rows,
        ))

    # Authentication
    auth = detect_auth_mechanisms(root, gitignore_patterns)
    sections.append("## Authentication & Security\n")
    if auth:
        for mechanism in auth:
            sections.append(f"- {mechanism}")
    else:
        sections.append("- No authentication mechanisms detected")
    sections.append("")

    # State Management
    state = detect_state_management(root, gitignore_patterns)
    if state["libraries"] or state["stores"]:
        sections.append("## State Management\n")
        if state["libraries"]:
            sections.append(f"- **Libraries:** {', '.join(state['libraries'])}")
        if state["stores"]:
            sections.append(f"- **Store files ({len(state['stores'])}):**")
            for store in state["stores"][:15]:
                sections.append(f"  - `{store['file']}`")
        sections.append("")

    # Real-time Features
    realtime = detect_realtime(root, gitignore_patterns)
    if realtime:
        sections.append("## Real-time Features\n")
        for rt in realtime:
            sections.append(f"- {rt}")
        sections.append("")

    # Accessibility
    a11y = analyze_accessibility(root, gitignore_patterns)
    if a11y["total_ui_files"] > 0:
        sections.append("## Accessibility Posture\n")
        sections.append(f"- **UI files with ARIA/role attributes:** {a11y['files_with_a11y']}/{a11y['total_ui_files']}")
        sections.append(f"- **aria-* attributes:** {a11y['aria_attributes']}")
        sections.append(f"- **role attributes:** {a11y['role_attributes']}")
        sections.append(f"- **tabIndex usage:** {a11y['tabindex_usage']}")
        sections.append(f"- **alt attributes:** {a11y['alt_attributes']}")
        if a11y["a11y_libraries"]:
            sections.append(f"- **A11y libraries:** {', '.join(a11y['a11y_libraries'])}")
        sections.append("")

    # Internationalization
    i18n = detect_i18n(root, gitignore_patterns)
    sections.append("## Internationalization\n")
    if i18n["libraries"]:
        sections.append(f"- **Libraries:** {', '.join(i18n['libraries'])}")
        if i18n["locale_dirs"]:
            sections.append(f"- **Locale directories:** {', '.join(i18n['locale_dirs'])}")
        sections.append(f"- **Locale files:** {i18n['locale_files']}")
    else:
        sections.append("- Not detected")
    sections.append("")

    # Feature Gaps
    gaps = detect_feature_gaps(root, gitignore_patterns, auth, i18n, a11y, realtime, state)
    sections.append("## Feature Gaps\n")
    if gaps:
        for gap in gaps:
            sections.append(f"- {gap}")
    else:
        sections.append("No significant feature gaps detected.")
    sections.append("")

    return "\n".join(sections)
