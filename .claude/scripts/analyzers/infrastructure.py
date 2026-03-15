"""Infrastructure & Architecture analyzer.

Detects build systems, CI/CD pipelines, containers, database schemas,
API endpoints, environment config, monitoring, and dependencies.
Language-agnostic, zero external dependencies.
"""

import json
import re
from pathlib import Path

from . import (
    classify_all_files,
    detect_ecosystems,
    detect_frameworks,
    detect_languages,
    find_package_jsons,
    load_gitignore_patterns,
    make_table,
    parse_package_json,
    read_file_safe,
    search_content,
    walk_source_files,
)

# ── Build System Detection ──────────────────────────────────────────────────

BUILD_SYSTEMS = [
    ("package.json", "npm/Node.js"),
    ("yarn.lock", "Yarn"),
    ("pnpm-lock.yaml", "pnpm"),
    ("bun.lockb", "Bun"),
    ("pyproject.toml", "Python (pyproject)"),
    ("setup.py", "Python (setuptools)"),
    ("setup.cfg", "Python (setuptools)"),
    ("Pipfile", "Python (Pipenv)"),
    ("poetry.lock", "Python (Poetry)"),
    ("uv.lock", "Python (uv)"),
    ("Cargo.toml", "Rust (Cargo)"),
    ("go.mod", "Go Modules"),
    ("go.sum", "Go Modules"),
    ("pom.xml", "Maven"),
    ("build.gradle", "Gradle"),
    ("build.gradle.kts", "Gradle (Kotlin DSL)"),
    ("Gemfile", "Ruby (Bundler)"),
    ("composer.json", "PHP (Composer)"),
    ("mix.exs", "Elixir (Mix)"),
    ("pubspec.yaml", "Dart/Flutter (Pub)"),
    ("CMakeLists.txt", "CMake"),
    ("Makefile", "Make"),
    ("Justfile", "Just"),
    ("Taskfile.yml", "Task"),
    ("Tiltfile", "Tilt"),
]


def detect_build_systems(root: Path) -> list[tuple[str, str]]:
    """Returns [(indicator_file, build_system_name)]."""
    found = []
    for indicator, name in BUILD_SYSTEMS:
        if (root / indicator).exists():
            found.append((indicator, name))
    return found


def extract_scripts(root: Path) -> dict[str, dict[str, str]]:
    """Extract scripts from all package.json files."""
    results = {}
    for pkg_path in find_package_jsons(root):
        data = parse_package_json(pkg_path)
        if data.get("scripts"):
            rel = str(pkg_path.relative_to(root))
            results[rel] = data["scripts"]
    return results


# ── CI/CD Detection ─────────────────────────────────────────────────────────

CI_INDICATORS = [
    (".github/workflows", "GitHub Actions"),
    (".gitlab-ci.yml", "GitLab CI"),
    ("Jenkinsfile", "Jenkins"),
    (".circleci", "CircleCI"),
    (".drone.yml", "Drone CI"),
    ("azure-pipelines.yml", "Azure Pipelines"),
    ("bitbucket-pipelines.yml", "Bitbucket Pipelines"),
    (".buildkite", "Buildkite"),
    (".travis.yml", "Travis CI"),
    ("appveyor.yml", "AppVeyor"),
    (".woodpecker.yml", "Woodpecker CI"),
]


def detect_ci_cd(root: Path) -> list[dict]:
    """Detect CI/CD systems and list their workflow files."""
    results = []
    for indicator, name in CI_INDICATORS:
        path = root / indicator
        if path.exists():
            entry = {"system": name, "path": indicator, "workflows": []}
            if path.is_dir():
                for f in sorted(path.iterdir()):
                    if f.suffix in (".yml", ".yaml"):
                        # Extract workflow name from file content
                        content = read_file_safe(f)
                        wf_name = f.stem
                        name_match = re.search(r'^name:\s*["\']?(.+?)["\']?\s*$', content, re.MULTILINE)
                        if name_match:
                            wf_name = name_match.group(1)
                        # Extract triggers
                        triggers = []
                        on_match = re.search(r'^on:\s*$', content, re.MULTILINE)
                        if on_match:
                            # Look for trigger types in the next few lines
                            after_on = content[on_match.end():]
                            for t in ["push", "pull_request", "schedule", "workflow_dispatch",
                                      "release", "issues", "merge_request"]:
                                if re.search(rf'^\s+{t}:', after_on, re.MULTILINE):
                                    triggers.append(t)
                        # Single-line on:
                        on_line = re.search(r'^on:\s*\[(.+)\]', content, re.MULTILINE)
                        if on_line:
                            triggers = [t.strip() for t in on_line.group(1).split(",")]
                        entry["workflows"].append({
                            "file": str(f.relative_to(root)),
                            "name": wf_name,
                            "triggers": triggers,
                        })
            else:
                entry["workflows"].append({"file": indicator, "name": name, "triggers": []})
            results.append(entry)
    return results


# ── Container Detection ─────────────────────────────────────────────────────

def detect_containers(root: Path) -> dict:
    """Detect Docker/container configuration."""
    dockerfiles = list(root.glob("Dockerfile*")) + list(root.glob("**/Dockerfile*"))
    # Filter out node_modules etc
    dockerfiles = [d for d in dockerfiles if ".git" not in str(d) and "node_modules" not in str(d)]

    compose_files = []
    for pattern in ["docker-compose*.yml", "docker-compose*.yaml", "compose*.yml", "compose*.yaml"]:
        compose_files.extend(root.glob(pattern))

    result = {"dockerfiles": [], "compose_files": []}

    for df in dockerfiles[:10]:  # limit
        content = read_file_safe(df)
        base_images = re.findall(r'^FROM\s+(\S+)', content, re.MULTILINE)
        exposed_ports = re.findall(r'^EXPOSE\s+(.+)', content, re.MULTILINE)
        stages = len(base_images)
        result["dockerfiles"].append({
            "path": str(df.relative_to(root)),
            "base_images": base_images,
            "exposed_ports": exposed_ports,
            "multi_stage": stages > 1,
            "stages": stages,
        })

    for cf in compose_files[:5]:
        content = read_file_safe(cf)
        # Extract service names (lines matching "  service_name:")
        services = re.findall(r'^  (\w[\w-]*):\s*$', content, re.MULTILINE)
        result["compose_files"].append({
            "path": str(cf.relative_to(root)),
            "services": services,
        })

    return result


# ── Database / Schema Detection ─────────────────────────────────────────────

def detect_database(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect database schemas and ORM usage."""
    result = {"orm": None, "models": [], "migrations": 0, "schema_files": []}

    # Prisma
    prisma_files = list(root.glob("**/schema.prisma")) + list(root.glob("**/*.prisma"))
    prisma_files = [f for f in prisma_files if "node_modules" not in str(f)]
    if prisma_files:
        result["orm"] = "Prisma"
        for pf in prisma_files:
            result["schema_files"].append(str(pf.relative_to(root)))
            content = read_file_safe(pf)
            models = re.findall(r'^model\s+(\w+)\s*\{', content, re.MULTILINE)
            enums = re.findall(r'^enum\s+(\w+)\s*\{', content, re.MULTILINE)
            result["models"].extend(models)
            result["enums"] = enums
            # Detect provider
            provider = re.search(r'provider\s*=\s*"(\w+)"', content)
            if provider:
                result["provider"] = provider.group(1)

    # Django models
    django_models = list(root.glob("**/models.py"))
    django_models = [f for f in django_models if "node_modules" not in str(f) and ".venv" not in str(f)]
    if django_models and not result["orm"]:
        result["orm"] = "Django ORM"
        for mf in django_models:
            content = read_file_safe(mf)
            models = re.findall(r'^class\s+(\w+)\s*\(.*Model.*\)', content, re.MULTILINE)
            result["models"].extend(models)
            result["schema_files"].append(str(mf.relative_to(root)))

    # SQLAlchemy
    sa_matches, sa_files = 0, 0
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext == ".py":
            content = read_file_safe(fpath)
            if "sqlalchemy" in content.lower() or "from sqlalchemy" in content:
                sa_matches += 1
    if sa_matches > 0 and not result["orm"]:
        result["orm"] = "SQLAlchemy"

    # TypeORM / Sequelize / Knex
    for orm_name, indicator in [("TypeORM", "typeorm"), ("Sequelize", "sequelize"), ("Knex", "knex"),
                                 ("Drizzle", "drizzle-orm"), ("Mongoose", "mongoose")]:
        pkg = root / "package.json"
        if pkg.exists():
            content = read_file_safe(pkg)
            if f'"{indicator}"' in content:
                if not result["orm"]:
                    result["orm"] = orm_name

    # Count migrations
    migration_dirs = list(root.glob("**/migrations"))
    migration_dirs = [d for d in migration_dirs if d.is_dir() and "node_modules" not in str(d)]
    for md in migration_dirs:
        result["migrations"] += len([f for f in md.iterdir() if f.is_file()])

    # SQL files
    sql_files = []
    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext == ".sql":
            sql_files.append(rel)
    if sql_files:
        result["sql_files"] = len(sql_files)

    return result


# ── API Endpoint Detection ──────────────────────────────────────────────────

# Regex patterns for different frameworks
API_PATTERNS = [
    # Express.js: router.get('/path', ...) or app.post('/path', ...)
    (r"(?:router|app)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['\"]([^'\"]+)['\"]",
     "Express"),
    # FastAPI / Flask: @app.get("/path") or @router.post("/path")
    (r"@(?:app|router)\.(get|post|put|patch|delete|options|head)\s*\(\s*['\"]([^'\"]+)['\"]",
     "FastAPI/Flask"),
    # Go net/http: http.HandleFunc("/path", ...)
    (r"(?:Handle|HandleFunc)\s*\(\s*['\"]([^'\"]+)['\"]",
     "Go net/http"),
    # Spring: @GetMapping("/path"), @PostMapping("/path")
    (r"@(Get|Post|Put|Patch|Delete)Mapping\s*\(\s*(?:value\s*=\s*)?['\"]([^'\"]+)['\"]",
     "Spring"),
    # Rails: get '/path', post '/path'
    (r"^\s*(get|post|put|patch|delete)\s+['\"]([^'\"]+)['\"]",
     "Rails"),
    # ASP.NET: [HttpGet("path")], [HttpPost("path")]
    (r"\[Http(Get|Post|Put|Patch|Delete)\s*\(\s*['\"]([^'\"]+)['\"]\s*\)\]",
     "ASP.NET"),
]


def detect_api_endpoints(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect API endpoints across the codebase."""
    endpoints: list[dict] = []
    framework = None
    source_exts = {".ts", ".js", ".tsx", ".jsx", ".py", ".go", ".java", ".kt", ".rb", ".cs", ".php"}

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        if ext not in source_exts:
            continue
        content = read_file_safe(fpath)
        if not content:
            continue

        for pattern, fw_name in API_PATTERNS:
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

    # Group by domain (first path segment)
    domains: dict[str, list[dict]] = {}
    for ep in endpoints:
        parts = ep["path"].strip("/").split("/")
        domain = parts[0] if parts and parts[0] else "root"
        domains.setdefault(domain, []).append(ep)

    return {"framework": framework, "endpoints": endpoints, "by_domain": domains}


# ── Environment Configuration ───────────────────────────────────────────────

def detect_environment(root: Path) -> dict:
    """Detect environment configuration files (never reads .env contents)."""
    env_files = []
    for f in root.iterdir():
        if f.name.startswith(".env") and f.is_file():
            env_files.append(f.name)

    # Count variables from .env.example (safe to read)
    example_vars = 0
    example = root / ".env.example"
    if example.exists():
        content = read_file_safe(example)
        example_vars = len([l for l in content.splitlines()
                           if l.strip() and not l.strip().startswith("#") and "=" in l])

    # Config directories
    config_dirs = []
    for d in ["config", "configs", "conf", ".config", "settings"]:
        if (root / d).is_dir():
            config_dirs.append(d)

    return {
        "env_files": sorted(env_files),
        "example_var_count": example_vars,
        "config_dirs": config_dirs,
    }


# ── Monitoring & Logging ────────────────────────────────────────────────────

LOGGING_LIBS = [
    ("winston", "Winston (Node.js)"),
    ("pino", "Pino (Node.js)"),
    ("bunyan", "Bunyan (Node.js)"),
    ("morgan", "Morgan (HTTP logger)"),
    ("log4js", "Log4js (Node.js)"),
    ("import logging", "Python logging"),
    ("from logging", "Python logging"),
    ("logrus", "Logrus (Go)"),
    ("zap", "Zap (Go)"),
    ("slog", "slog (Go)"),
    ("log4j", "Log4j (Java)"),
    ("slf4j", "SLF4J (Java)"),
    ("logback", "Logback (Java)"),
    ("NLog", "NLog (.NET)"),
    ("Serilog", "Serilog (.NET)"),
]

MONITORING_LIBS = [
    ("sentry", "Sentry"),
    ("@sentry/", "Sentry"),
    ("newrelic", "New Relic"),
    ("datadog", "Datadog"),
    ("prometheus", "Prometheus"),
    ("opentelemetry", "OpenTelemetry"),
    ("@opentelemetry/", "OpenTelemetry"),
    ("elastic-apm", "Elastic APM"),
    ("grafana", "Grafana"),
    ("statsd", "StatsD"),
]


def detect_monitoring(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect logging and monitoring libraries."""
    logging_found = set()
    monitoring_found = set()

    for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
        content = read_file_safe(fpath)
        if not content:
            continue
        content_lower = content.lower()
        for indicator, name in LOGGING_LIBS:
            if indicator.lower() in content_lower:
                logging_found.add(name)
        for indicator, name in MONITORING_LIBS:
            if indicator.lower() in content_lower:
                monitoring_found.add(name)

    return {
        "logging": sorted(logging_found),
        "monitoring": sorted(monitoring_found),
    }


# ── Dependency Summary ──────────────────────────────────────────────────────

def summarize_dependencies(root: Path) -> list[dict]:
    """Summarize dependencies from all package manifests."""
    results = []
    for pkg_path in find_package_jsons(root):
        data = parse_package_json(pkg_path)
        rel = str(pkg_path.relative_to(root))
        results.append({
            "manifest": rel,
            "name": data.get("name", ""),
            "production": len(data.get("dependencies", [])),
            "dev": len(data.get("devDependencies", [])),
            "workspaces": data.get("workspaces", []),
        })

    # Python
    for fname in ["pyproject.toml", "requirements.txt", "Pipfile"]:
        fpath = root / fname
        if fpath.exists():
            content = read_file_safe(fpath)
            if fname == "requirements.txt":
                deps = len([l for l in content.splitlines() if l.strip() and not l.startswith("#")])
                results.append({"manifest": fname, "name": "", "production": deps, "dev": 0})
            elif fname == "pyproject.toml":
                # Simple count of dependencies lines
                dep_section = re.search(r'\[project\].*?dependencies\s*=\s*\[(.*?)\]',
                                       content, re.DOTALL)
                deps = 0
                if dep_section:
                    deps = len([l for l in dep_section.group(1).splitlines()
                               if l.strip() and l.strip() != '"' and not l.strip().startswith("#")])
                results.append({"manifest": fname, "name": "", "production": deps, "dev": 0})

    # Cargo.toml
    cargo = root / "Cargo.toml"
    if cargo.exists():
        content = read_file_safe(cargo)
        deps = len(re.findall(r'^\w[\w-]*\s*=', content, re.MULTILINE))
        results.append({"manifest": "Cargo.toml", "name": "", "production": deps, "dev": 0})

    # go.mod
    gomod = root / "go.mod"
    if gomod.exists():
        content = read_file_safe(gomod)
        deps = len(re.findall(r'^\t\S+', content, re.MULTILINE))
        results.append({"manifest": "go.mod", "name": "", "production": deps, "dev": 0})

    return results


# ── Cross-Cutting Concerns ──────────────────────────────────────────────────

def detect_crosscutting(root: Path, gitignore_patterns: list[str]) -> dict:
    """Detect middleware, auth, CORS, rate limiting, etc."""
    concerns = {}

    checks = [
        ("Rate Limiting", [r"rate.?limit", r"express-rate-limit", r"throttle"]),
        ("CORS", [r"cors\b", r"Access-Control-Allow"]),
        ("CSRF Protection", [r"csrf", r"csurf", r"csrftoken"]),
        ("Helmet / Security Headers", [r"helmet\b", r"security.?headers"]),
        ("Authentication Middleware", [r"auth.?middleware", r"passport\b", r"jwt.?middleware"]),
        ("Request Validation", [r"express-validator", r"joi\b", r"zod\b", r"yup\b", r"class-validator"]),
        ("Compression", [r"compression\b", r"gzip"]),
        ("Health Check", [r"/health", r"healthcheck", r"/readiness", r"/liveness"]),
    ]

    for name, patterns in checks:
        for pattern in patterns:
            matches, files = 0, 0
            for rel, fpath, ext, _ in walk_source_files(root, gitignore_patterns):
                content = read_file_safe(fpath)
                if content and re.search(pattern, content, re.IGNORECASE):
                    matches += 1
            if matches:
                concerns[name] = "detected"
                break
        else:
            concerns[name] = "not detected"

    return concerns


# ── Report Generator ────────────────────────────────────────────────────────

def generate_report(root: Path, gitignore_patterns: list[str] | None = None) -> str:
    """Generate the full infrastructure & architecture report."""
    root = root.resolve()
    if gitignore_patterns is None:
        gitignore_patterns = load_gitignore_patterns(root)

    sections = []
    sections.append("# Infrastructure & Architecture Report\n")
    sections.append(f"> Auto-generated static analysis for `{root.name}`\n")

    # Languages
    languages = detect_languages(root, gitignore_patterns)
    ecosystems = detect_ecosystems(root, gitignore_patterns)
    if languages:
        sections.append("## Languages & Ecosystems\n")
        top_langs = list(languages.items())[:10]
        sections.append(make_table(
            ["Language", "Files"],
            [[lang, str(count)] for lang, count in top_langs],
        ))
        primary_eco = list(ecosystems.keys())[0] if ecosystems else "Unknown"
        sections.append(f"**Primary ecosystem:** {primary_eco}\n")

    # Frameworks
    frameworks = detect_frameworks(root)
    if frameworks:
        sections.append("## Frameworks Detected\n")
        for fw in frameworks:
            sections.append(f"- {fw}")
        sections.append("")

    # Build System
    build_systems = detect_build_systems(root)
    if build_systems:
        sections.append("## Build System\n")
        sections.append(make_table(
            ["Indicator", "System"],
            [[ind, sys] for ind, sys in build_systems],
        ))

    # Scripts
    scripts = extract_scripts(root)
    if scripts:
        sections.append("## Build Scripts\n")
        for manifest, cmds in scripts.items():
            sections.append(f"### `{manifest}`\n")
            sections.append(make_table(
                ["Script", "Command"],
                [[name, cmd[:80]] for name, cmd in sorted(cmds.items())],
            ))

    # Monorepo / Workspaces
    root_pkg = parse_package_json(root / "package.json")
    if root_pkg.get("workspaces"):
        sections.append("## Monorepo Workspaces\n")
        for ws in root_pkg["workspaces"]:
            sections.append(f"- `{ws}`")
        sections.append("")

    # CI/CD
    ci_systems = detect_ci_cd(root)
    if ci_systems:
        sections.append("## CI/CD Pipelines\n")
        for ci in ci_systems:
            sections.append(f"### {ci['system']}\n")
            if ci["workflows"]:
                sections.append(make_table(
                    ["File", "Name", "Triggers"],
                    [[wf["file"], wf["name"], ", ".join(wf["triggers"]) or "—"]
                     for wf in ci["workflows"]],
                ))

    # Containers
    containers = detect_containers(root)
    if containers["dockerfiles"] or containers["compose_files"]:
        sections.append("## Containerization\n")
        if containers["dockerfiles"]:
            sections.append("### Dockerfiles\n")
            sections.append(make_table(
                ["File", "Base Images", "Multi-stage", "Exposed Ports"],
                [[d["path"], ", ".join(d["base_images"]), "Yes" if d["multi_stage"] else "No",
                  ", ".join(d["exposed_ports"]) or "—"] for d in containers["dockerfiles"]],
            ))
        if containers["compose_files"]:
            sections.append("### Compose Files\n")
            for cf in containers["compose_files"]:
                sections.append(f"**{cf['path']}** — services: {', '.join(cf['services']) or '(none)'}\n")

    # Database
    db = detect_database(root, gitignore_patterns)
    if db["orm"] or db["models"] or db.get("sql_files"):
        sections.append("## Database\n")
        if db["orm"]:
            sections.append(f"- **ORM:** {db['orm']}")
        if db.get("provider"):
            sections.append(f"- **Provider:** {db['provider']}")
        if db["models"]:
            sections.append(f"- **Models ({len(db['models'])}):** {', '.join(db['models'])}")
        if db.get("enums"):
            sections.append(f"- **Enums ({len(db['enums'])}):** {', '.join(db['enums'])}")
        if db["migrations"]:
            sections.append(f"- **Migrations:** {db['migrations']} files")
        if db["schema_files"]:
            sections.append(f"- **Schema files:** {', '.join(db['schema_files'])}")
        if db.get("sql_files"):
            sections.append(f"- **SQL files:** {db['sql_files']}")
        sections.append("")

    # API Endpoints
    api = detect_api_endpoints(root, gitignore_patterns)
    if api["endpoints"]:
        sections.append("## API Endpoints\n")
        sections.append(f"- **Framework:** {api['framework'] or 'Unknown'}")
        sections.append(f"- **Total endpoints:** {len(api['endpoints'])}\n")

        # Summary by domain
        domain_rows = []
        for domain, eps in sorted(api["by_domain"].items()):
            methods = {}
            for ep in eps:
                methods[ep["method"]] = methods.get(ep["method"], 0) + 1
            domain_rows.append([
                domain,
                str(methods.get("GET", 0)),
                str(methods.get("POST", 0)),
                str(methods.get("PUT", 0)),
                str(methods.get("DELETE", 0)),
                str(len(eps)),
            ])
        sections.append(make_table(
            ["Domain", "GET", "POST", "PUT", "DELETE", "Total"],
            domain_rows,
        ))

    # Environment
    env = detect_environment(root)
    if env["env_files"] or env["config_dirs"]:
        sections.append("## Environment Configuration\n")
        if env["env_files"]:
            sections.append(f"- **Env files:** {', '.join(env['env_files'])}")
        if env["example_var_count"]:
            sections.append(f"- **Variables in .env.example:** {env['example_var_count']}")
        if env["config_dirs"]:
            sections.append(f"- **Config directories:** {', '.join(env['config_dirs'])}")
        sections.append("")

    # Monitoring & Logging
    monitoring = detect_monitoring(root, gitignore_patterns)
    sections.append("## Monitoring & Logging\n")
    if monitoring["logging"]:
        sections.append(f"- **Logging:** {', '.join(monitoring['logging'])}")
    else:
        sections.append("- **Logging:** No structured logging library detected")
    if monitoring["monitoring"]:
        sections.append(f"- **Monitoring/APM:** {', '.join(monitoring['monitoring'])}")
    else:
        sections.append("- **Monitoring/APM:** Not detected")
    sections.append("")

    # Dependencies
    deps = summarize_dependencies(root)
    if deps:
        sections.append("## Dependency Summary\n")
        sections.append(make_table(
            ["Manifest", "Name", "Production", "Dev"],
            [[d["manifest"], d["name"], str(d["production"]), str(d["dev"])] for d in deps],
        ))

    # Cross-Cutting Concerns
    concerns = detect_crosscutting(root, gitignore_patterns)
    sections.append("## Cross-Cutting Concerns\n")
    sections.append(make_table(
        ["Concern", "Status"],
        [[name, status] for name, status in sorted(concerns.items())],
    ))

    # Architecture summary
    file_roles = classify_all_files(root, gitignore_patterns)
    sections.append("## Architecture Pattern Summary\n")
    role_counts = {role: len(files) for role, files in sorted(file_roles.items()) if role != "other"}
    if role_counts:
        sections.append(make_table(
            ["Layer/Role", "Files"],
            [[role, str(count)] for role, count in sorted(role_counts.items(), key=lambda x: -x[1])],
        ))

    # Gaps
    sections.append("## Identified Gaps\n")
    gaps = []
    if not monitoring["logging"]:
        gaps.append("No structured logging library detected")
    if not monitoring["monitoring"]:
        gaps.append("No monitoring/APM solution detected")
    if concerns.get("Health Check") == "not detected":
        gaps.append("No health check endpoint found")
    if concerns.get("Rate Limiting") == "not detected":
        gaps.append("No rate limiting detected")
    if concerns.get("CSRF Protection") == "not detected":
        gaps.append("No CSRF protection detected")
    if not containers["dockerfiles"]:
        gaps.append("No Dockerfile found — containerization not configured")
    if not ci_systems:
        gaps.append("No CI/CD pipeline detected")
    if not db["orm"] and not db.get("sql_files"):
        gaps.append("No database/ORM detected")

    if gaps:
        for gap in gaps:
            sections.append(f"- {gap}")
    else:
        sections.append("No significant gaps detected.")
    sections.append("")

    return "\n".join(sections)
