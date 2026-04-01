# Arsenale Agent Guide

## Purpose
Use `tools/arsenale-cli` as the primary operator and smoke-test client for this platform. Prefer it over ad hoc `curl` when you want to verify behavior end-to-end.

## Build And Verify
Before relying on the CLI, build it from the repo root:

```bash
go test ./tools/arsenale-cli/...
go build -o /tmp/arsenale-cli ./tools/arsenale-cli
```

For the local dev stack, point the CLI at `https://localhost:3000`:

```bash
/tmp/arsenale-cli --server https://localhost:3000 health
/tmp/arsenale-cli --server https://localhost:3000 login
```

The CLI stores credentials in `~/.arsenale/config.yaml`. The config defaults are:

```bash
/tmp/arsenale-cli config
/tmp/arsenale-cli config get server_url
/tmp/arsenale-cli config set server_url https://localhost:3000
```

## Test Flow
Use this sequence when checking the platform after a change:

1. `arsenale health` to confirm the API is reachable.
2. `arsenale login --server https://localhost:3000` to refresh local credentials.
3. `arsenale whoami` to confirm the authenticated tenant/user context.
4. `arsenale connection list` and `arsenale gateway list` to verify the resource layer.
5. `arsenale session list` and `arsenale gateway instances <id>` to verify runtime state.
6. `arsenale gateway test <id>` before trying a manual `arsenale connect ssh <name>` or `arsenale connect rdp <name>`.
7. Use `-o json` for machine checks and `--quiet` when only IDs matter.

For gateway and session debugging, these commands are especially useful:

```bash
/tmp/arsenale-cli --server https://localhost:3000 gateway tunnel-overview
/tmp/arsenale-cli --server https://localhost:3000 gateway instances <gateway-id>
/tmp/arsenale-cli --server https://localhost:3000 session count
/tmp/arsenale-cli --server https://localhost:3000 rdgw status
```

## Alignment Rule
Any change that affects API routes, response fields, auth flows, config defaults, server URLs, tenant selection, or deployment wiring must be reflected in `tools/arsenale-cli` in the same change set.

That means:

1. Update the CLI command or output handling when backend contracts change.
2. Rebuild and retest the CLI against the current stack.
3. Treat CLI help output and smoke tests as acceptance criteria, not an afterthought.

If the platform changes and the CLI is not updated to match, the change is incomplete.

## Practical Scope
The most commonly used CLI entry points are:

- `health`
- `login`
- `whoami`
- `config`
- `connection`
- `gateway`
- `session`
- `rdgw`
- `vault`
- `connect`

Use `arsenale [command] --help` before assuming flag names or subcommand availability.
