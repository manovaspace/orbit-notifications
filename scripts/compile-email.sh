#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/pkg/mailtemplates/compiled"
mkdir -p "$OUT"
cd "$ROOT"
for name in otp_login invite_developer owner_challenge; do
  src="$ROOT/templates/email/${name}.mjml"
  if [[ -f "$src" ]]; then
    bunx mjml@4 "$src" -o "$OUT/${name}.html" --config.minify false
  fi
done
for f in "$ROOT/templates/email/"*.txt; do
  [[ -f "$f" ]] && cp "$f" "$OUT/"
done
