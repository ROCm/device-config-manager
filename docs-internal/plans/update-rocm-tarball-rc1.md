# Plan: Update ROCm Tarball URL to 7.14.0rc1

## Goal

Replace the hardcoded `7.14.0rc0` tarball URL and version string with `7.14.0rc1` in the two
canonical definition sites. All downstream consumers inherit via `?=` overrides and `export`.

## Success criteria

- `dev.env` and `Makefile` each contain the new URL and version string exactly once.
- No other file references the old `rc0` URL (verify with `grep -r rc0`).
- Branch `update-rocm-tarball-url-rc1` ready for PR against `main`.

## Changes

| File | Line | Old value | New value |
|------|------|-----------|-----------|
| `dev.env` | 25 | `therock-dist-linux-multiarch-7.14.0rc0.tar.gz` | `therock-dist-linux-multiarch-7.14.0rc1.tar.gz` |
| `dev.env` | 26 | `ROCM_VERSION ?= 7.14.0rc0` | `ROCM_VERSION ?= 7.14.0rc1` |
| `Makefile` | 128 | `therock-dist-linux-multiarch-7.14.0rc0.tar.gz` | `therock-dist-linux-multiarch-7.14.0rc1.tar.gz` |
| `Makefile` | 129 | `ROCM_VERSION ?= 7.14.0rc0` | `ROCM_VERSION ?= 7.14.0rc1` |


## Steps

1. Edit `dev.env` lines 25–26.
2. Edit `Makefile` lines 128–129.
3. Verify: `grep -rn "rc0\|7.14.0rc" dev.env Makefile` — should only show rc1.
4. Commit: `git commit -m "chore(rocm): bump tarball URL to 7.14.0rc1 (gfx110X-all)"`.

## Risks

- Low. `?=` semantics mean anyone overriding `ROCM_TARBALL_URL` externally is unaffected.
- The version string change (`rc0` → `rc1`) must be consistent between URL and `ROCM_VERSION`
  or the extracted directory `/opt/rocm-${ROCM_VERSION}/` will not match the tarball.
