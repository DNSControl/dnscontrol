# Colors support

DNSControl is using colors on both its console and notification outputs, to denote changes, such as during preview or push events.

This is on by default in all interactive prompts supporting colors. See below on how to bypass it.

## Disable colors

There are cases that we might need to disable colored output, such as in certain CI environments or particular notification mechanisms that don't support ANSI color codes.

In order to do so, a global `--no-colors` command option is provided, which when set `--no-colors=true`, will disable colors globally.

Alternatively, a `NO_COLOR` environment variable set to any non-empty string will disable color output.

## (Force) Enable colors

If color support is not correctly detected, providing `--no-colors=false` would force-enable coloring, bypassing modules TTY detection.

## Force colors in CI environments

Some CI environments, while supporting colors, fail autodetection, such as GHA. In that case, colors can be forcibly enabled via `--no-colors=false` or by setting a `CI` env variable to `true` (if not autoset).
