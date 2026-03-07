# Third-party customization

This document describes the customization surface available to institutions running
their own bbl installation. The design intent is to make the common cases easy and
declarative while keeping the door open for deeper customization — without
rearchitecting.

## Principles

- **Simplicity first.** Start small; growth should not require rearchitecting.
- **Declarative over programmatic.** Customization through config and i18n files,
  not code.
- **Go knowledge is an acceptable ceiling.** Simple branding, translation, new work
  kinds, and most validation rules require no programming. Only truly structural
  extensions — new field types, new integrations, new auth providers — require Go.
  That is a deliberate and acceptable tradeoff.
- **Deploy and configure are separate concerns.** Code upgrades (new binary),
  profile changes, and i18n/branding updates (normal deploy) each happen on their
  own schedule.

---

## Layers of customization

| Layer | What changes | Who does it | How |
|---|---|---|---|
| Branding | Logo, colors, institution name | Anyone | Config + static file override |
| Translations & labels | All UI text, field labels, help, guidelines | Translator | i18n locale files |
| Profile | Work kinds, active fields, required/optional/locked, validation rules | Content specialist | Edit profile YAML + deploy |
| Advanced | New field types, new integrations, new auth providers | Go developer | Go source + bbl release |

---

## Override file discovery (simple customization)

The bbl binary embeds all defaults (templates, assets, base locale). At startup it
looks for override files at well-known paths relative to a configured `assets_dir`
(defaults to `./assets`). If an override file is present it is used; otherwise the
embedded default is used. No recompilation, no special tooling.

```
assets/
  logo.svg        # replaces embedded logo
  custom.css      # injected after base stylesheet
  locales/
    en.yaml       # overrides or extends embedded English strings
    nl.yaml       # adds Dutch locale
```

All override discovery happens at startup. Changes take effect after a restart.

## Branding

**Institution name** — set in the main config file. Used in page titles, emails, and
OAI-PMH identity responses.

**Logo** — place `assets/logo.svg` (or `.png`). Falls back to the embedded bbl logo
if absent.

**Colors and typography** — `assets/custom.css` is injected after the base
stylesheet. Use CSS custom properties to override design tokens:

```css
:root {
  --color-primary:    #003366;
  --color-accent:     #E87722;
  --font-family-body: "Source Sans Pro", sans-serif;
}
```

No build step needed.

---

## i18n and labels

All human-readable text lives in locale files, not in Go source or the profile. This
includes:

- UI chrome (navigation, buttons, status labels)
- **Field labels and help text** — keyed off profile field names
- **Submission guidelines** — per-kind free text shown during the submission flow
- **Work kind display names and descriptions**

The profile is the structural authority (what fields exist, their canonical key
names). The i18n files are the presentational authority (what those fields are called,
how they are explained). This is a clean split: changing a label never requires a
profile apply; changing which fields exist never touches i18n files directly.

### Key convention

i18n keys follow the structure of the profile:

```
work.<kind>.label                       # display name for a work kind
work.<kind>.guidelines                  # submission guidelines (markdown)
work.<kind>.field.<field>.label         # field label
work.<kind>.field.<field>.hint          # optional short help text below the input
work.<kind>.field.<field>.placeholder   # input placeholder text
```

Example:
```yaml
# locales/en.yaml
work:
  journal_article:
    label: Journal article
    guidelines: |
      Please provide the full journal title, volume, and issue number.
      DOI is strongly recommended.
    field:
      volume:
        label: Volume
        hint: The volume number of the journal issue
      issue:
        label: Issue
```

A locale file missing a key falls back to the default locale (`en`). Institutions
provide only the keys they want to override or translate; everything else is
inherited from the upstream default locale.

### Updating i18n

i18n files are loaded at startup and held in memory. Changes take effect after a
restart. No special apply command is needed — i18n changes are presentational and
have no data implications, so they follow the normal deploy cycle.

---

## Profile customization

The structural layer. Defines which work kinds are active, which fields are enabled
per kind, and how they behave (required, optional, locked). Profile keys serve as the
canonical identifiers that i18n files reference.

See `greenfield-schema-sketch.md → Field model and profiles` for the full design,
including the `bbl profile diff` safety check and change classification.

### What can be customized

- Which work kinds are active (and which are deprecated)
- Defining new work kinds — any combination of fields from the Go field catalog
- Which fields are active per kind, and their required/optional status
- Field display order (declaration order in YAML)
- Which fields are locked (protected from harvester overwrite)
- Validation rules — constraints beyond required/optional expressed as CEL expressions
  in the profile (e.g. allowed values, max length, cross-field rules) *(planned;
  profiles without rules work unchanged)*

### What cannot be customized via profile

- Field labels, help text, or submission guidelines — those live in i18n
- Adding entirely new field *types* not present in the Go model — requires a Go change
  and a bbl release
- Changing field semantics or structural schema

---

## Advanced customization — custom binary (Go required)

For the full technical reference on extension interfaces, registries, and hooks,
see `extension-points.md`.


For changes that go beyond file overrides, institutions build their own binary that
imports bbl as a Go library. This avoids forks: bbl is tracked as a normal Go module
dependency and upgraded like any other library. Breaking changes surface at compile
time.

```go
// cmd/mybbl/main.go
package main

import (
    "github.com/ugent-library/bbl"
    "github.com/myinstitution/mybbl/kinds"
    "github.com/myinstitution/mybbl/integrations"
)

func main() {
    app := bbl.New(bbl.Config{ /* ... */ })
    app.RegisterWorkKind(kinds.Dataset)
    app.RegisterIntegration(integrations.PureHarvester)
    app.Run()
}
```

What can be registered this way:

- **New field types** — add to Go model, register with app; then usable in profiles
- **New integrations** — harvesters, exporters, authentication providers
- **Role additions** — built-in roles (admin, curator, submitter, reviewer) cover
  most cases; extra roles registered at startup

The registration API is the stable extension surface. bbl maintains backwards
compatibility on registered interfaces across minor versions.

**Note on plugins**: Go's runtime plugin mechanism (`plugin` package) is fragile,
platform-limited, and requires exact version alignment between host and plugin. The
custom binary pattern is the idiomatic Go alternative: compile-time composition gives
the same extensibility without the runtime fragility. There is no dynamic plugin
system.

---

## Open questions

- **Harvester source trust**: can third parties register new sources and configure
  trust priority without a schema migration?
- **Role customization**: are the four built-in roles sufficient for all anticipated
  third-party use cases?
