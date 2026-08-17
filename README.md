# mailwizz-cli

A command line client for the [MailWizz](https://www.mailwizz.com/) API, built so both humans and
LLM-based tools can drive a MailWizz instance from the terminal.

The command tree mirrors the MailWizz API's resource structure (`mailwizz-cli lists subscribers
create`, `mailwizz-cli campaigns stats`, ...), every command and flag is documented via `--help`, and
every response is printed as indented JSON so it is easy for both a person and a script to read.

The API client and the CLI command tree are **generated** from MailWizz's OpenAPI schema
(`openapi/schema.json`). When MailWizz publishes a new version of that schema, replace the file,
run `task generate`, and the CLI picks up the change.

## Installation

### Homebrew (macOS and Linux)

```sh
brew tap onetwist-software/mailwizz-cli
brew install --cask mailwizz-cli
```

### Prebuilt binaries

Every [release](https://github.com/onetwist-software/mailwizz-cli/releases) publishes binaries for
macOS, Linux, and Windows, on amd64 and arm64. Download the archive for your platform, extract it, and
put `mailwizz-cli` on your `PATH`.

### Using Go

Requires Go 1.25 or newer.

```sh
go install github.com/onetwist-software/mailwizz-cli/cmd/mailwizz-cli@latest
```

### From source

```sh
git clone https://github.com/onetwist-software/mailwizz-cli.git
cd mailwizz-cli
task build      # writes ./bin/mailwizz-cli
```

## Quick start

Configure your MailWizz API URL and API key once:

```sh
mailwizz-cli config set --api-url https://your-domain.com/api --api-key YOUR_API_KEY
```

This writes `~/.mailwizz-cli.json` (with `0600` permissions, since it holds your API key). Check what
is currently configured with:

```sh
mailwizz-cli config get
```

Then use any command:

```sh
mailwizz-cli lists list
mailwizz-cli lists create \
  --general-name "My List" \
  --general-description "A list for my newsletter" \
  --defaults-from-name "Jane Doe" \
  --defaults-from-email jane@example.com \
  --defaults-reply-to jane@example.com

mailwizz-cli lists subscribers create \
  --list-uid LIST_UID \
  --email subscriber@example.com \
  --fname Jane

mailwizz-cli campaigns stats --campaign-uid CAMPAIGN_UID
mailwizz-cli campaigns pause-unpause --campaign-uid CAMPAIGN_UID
```

Every command and flag is discoverable:

```sh
mailwizz-cli --help
mailwizz-cli lists --help
mailwizz-cli lists subscribers create --help
```

## Configuration

mailwizz-cli resolves its API URL and API key in this order:

1. Environment variables `MAILWIZZ_API_URL` and `MAILWIZZ_API_KEY`, if set.
2. `~/.mailwizz-cli.json`, written by `mailwizz-cli config set`.

There are no `--api-url`/`--api-key` flags on individual commands; this keeps `--help` output focused
on what each command actually does. Use the environment variables if you need to override the config
file temporarily (for example in CI or when scripting against a second MailWizz instance).

## Output and exit codes

Every command prints the MailWizz API's JSON response, pretty-printed, to stdout. On a 2xx response
the process exits `0`. On any other status, the same JSON body (including MailWizz's own error
message) is printed and the process exits `1` — there is no separate, redundant "error:" line, so
scripts and LLMs can rely on stdout alone to see what happened.

Flags with a fixed set of allowed values (as declared by the MailWizz schema, e.g. subscriber status)
are validated locally before any request is sent, so invalid input fails fast with a clear message
instead of a round trip to the API.

## Command reference

Commands are grouped by resource, matching the MailWizz API:

| Command | Covers |
|---|---|
| `mailwizz-cli config` | `set`, `get` — manage the local configuration file |
| `mailwizz-cli campaigns` | CRUD, `copy`, `mark-sent`, `pause-unpause`, `stats`, `bounces`, `complaints`, `delivery-logs`, `email-message-id`, `tracking`, `unsubscribes` |
| `mailwizz-cli lists` | CRUD, `copy`, `fields`, `segments`, `subscribers` |
| `mailwizz-cli lists fields` | CRUD, `types` |
| `mailwizz-cli lists segments` | CRUD, `condition-operators` |
| `mailwizz-cli lists subscribers` | CRUD, `search-by-email`, `search-by-custom-fields`, `search-by-email-in-all-lists`, `unsubscribe`, `unsubscribe-by-email-from-all-lists` |
| `mailwizz-cli templates` | CRUD |
| `mailwizz-cli transactional-emails` | `list`, `view`, `create`, `delete` |
| `mailwizz-cli customers` | `create` |
| `mailwizz-cli countries` | `list`, `zones` |
| `mailwizz-cli delivery-servers` | `list`, `view` |

Run `mailwizz-cli <command> --help` (at any depth) for the exact flags, descriptions, and which ones
are required — that help text is generated from the same schema as the commands themselves, so it
never drifts out of date.

## How it's built

```
cmd/
  mailwizz-cli/       CLI entry point: wires hand-written and generated commands together
  mailwizz-cli-gen/   code generator entry point (run via `task generate`)
internal/
  apiclient/          low level HTTP client: auth header, query/multipart encoding
  config/             ~/.mailwizz-cli.json and MAILWIZZ_API_URL / MAILWIZZ_API_KEY handling
  output/             prints API responses, maps HTTP status to process exit codes
  commands/           hand-written commands (currently just "config") and the ResolveClient helper
  generated/
    api/              generated: typed request structs + one Client method per MailWizz operation
    cli/              generated: the full *cli.Command tree with flags and actions
generator/
  parser.go           loads and resolves openapi/schema.json (via kin-openapi)
  model.go, naming.go, defaults.go, tree.go, render.go   the internal model and naming rules
  overrides.go, overrides.yaml   hand-maintained corrections (see below)
  gen_api.go, gen_cli.go, generate.go   renders and writes internal/generated/**
openapi/
  schema.json         source of truth: MailWizz's OpenAPI schema
```

### Why generated code is committed

`internal/generated/**` is committed to the repository (not built on the fly), so the project builds
with a plain `go build`/`go install` and generated code is reviewable in pull requests like any other
change. `task generate:check` (used by CI) regenerates and fails the build if that produces a diff,
which catches a schema/overrides change that someone forgot to regenerate for.

### The overrides file

`openapi/schema.json` is authoritative, but it has two limitations `generator/overrides.yaml` fixes
up:

- **Naming**: most operations fit a simple rule (a `GET` without a trailing path parameter becomes
  `list`, a `GET` with one becomes `view`, `POST`/`PUT`/`DELETE` become `create`/`update`/`delete`).
  A handful of "action" endpoints (`copy`, `pause-unpause`, `stats`, tracking, `search-by-*`, ...)
  read better as an explicit command name than as a generic verb nested one level deeper than
  necessary, so those get an explicit entry.
- **Pagination**: MailWizz's real API accepts `page`/`per_page` on every collection `GET` endpoint,
  but schema.json doesn't declare them. The overrides file adds them where they belong.

If you update `openapi/schema.json` and a new operation doesn't fit the default naming rule, add an
entry to `generator/overrides.yaml` (loading it fails fast with a clear error if it ever references an
operation that no longer exists, so a rename is caught immediately).

### Regenerating after a schema update

```sh
task generate      # writes internal/generated/api and internal/generated/cli
task check          # generate:check + lint + test + build, same as CI
```

## Development

This project uses [Task](https://taskfile.dev/) for common commands:

```sh
task build      # build ./bin/mailwizz-cli
task test       # go test -race with coverage
task lint       # golangci-lint (see .golangci.yml)
task generate   # regenerate internal/generated/** from openapi/schema.json
task check      # everything CI runs
```

`.golangci.yml` enables effectively every golangci-lint linter; the small number that are disabled are
each documented inline with why (deprecated, or fundamentally at odds with idiomatic Go for this
codebase).

See [agents.md](agents.md) for repository conventions aimed at AI coding agents (and a good summary
for human contributors too).

## Releasing

Releases are built with [GoReleaser](https://goreleaser.com/) (`.goreleaser.yaml`), producing
binaries for macOS, Linux, and Windows on amd64 and arm64.

- `.github/workflows/ci.yml` runs on every push/PR: regenerates and diffs `internal/generated/**`,
  lints, tests, and builds.
- `.github/workflows/release.yml` runs GoReleaser whenever a tag matching `v*` is pushed (e.g. `git
  tag v0.1.0 && git push origin v0.1.0`), publishing archives, checksums, and a changelog to a GitHub
  Release, and pushing an updated Homebrew Cask to
  [onetwist-software/homebrew-mailwizz-cli](https://github.com/onetwist-software/homebrew-mailwizz-cli).
- `task release:check` validates `.goreleaser.yaml` locally.
- `task release:snapshot` builds every platform/arch locally into `dist/` without publishing, useful
  for testing a packaging change before tagging a release.

## Testing

- `generator/*_test.go`: unit tests for the naming/flattening rules, the OpenAPI parser, overrides,
  and the resource tree builder, plus golden-file tests that regenerate from both a small fixture
  schema and the real `openapi/schema.json` (the latter doubles as a check that the committed
  `internal/generated/**` is not stale).
- `internal/apiclient`, `internal/config`, `internal/output`: unit tests, including
  `httptest.Server`-backed tests for the HTTP client.
- `internal/commands`, `cmd/mailwizz-cli`: end-to-end tests that invoke the real root command against
  a mock MailWizz server, covering list/subscriber CRUD, bracketed and required request fields, query
  parameters, an overridden command name, enum flag validation, and the API-error output path.

Run everything with `task test`.

## License

MIT, see [LICENSE](LICENSE).
