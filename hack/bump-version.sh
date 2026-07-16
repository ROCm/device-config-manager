#!/bin/bash
#
# Copyright (c) Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Standalone version bump for release-branch creation. Rewrites version
# strings across docs/, helm-charts/, Makefile, and example/ for a DCM
# release. Does not touch git (no branch/commit/tag) — intended for CICD
# bootstrap use by the Prepare Branch tool. Idempotent: re-running with the
# same version produces no further diff.
#
# usage:
#   hack/bump-version.sh v1.5.0        # (or 1.5.0 — leading v optional)

set -euo pipefail

usage() {
	echo "usage: $0 <version>" >&2
	echo "  e.g. $0 v1.5.0  (or 1.5.0)" >&2
	exit 1
}

[ $# -eq 1 ] || usage

RAW_VERSION="$1"

if [[ "$RAW_VERSION" =~ ^v?([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
	VERSION_NUM="${BASH_REMATCH[1]}"
	VERSION_TAG="v${VERSION_NUM}"
else
	echo "error: invalid version '$RAW_VERSION' (expected vX.Y.Z or X.Y.Z)" >&2
	usage
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

CHANGED_FILES=()

require_file() {
	if [ ! -f "$1" ]; then
		echo "error: expected file not found: $1" >&2
		exit 1
	fi
}

track_change() {
	CHANGED_FILES+=("$1")
}

# Insert an idempotent placeholder release-notes section for $ver right after
# the file's top-level heading line, unless that section already exists.
# Matches on heading text (not line number) so a preamble before the heading
# doesn't misplace the insert.
insert_releasenotes_placeholder() {
	local file="$1" heading="$2" ver="$3"
	if grep -q "^## ${ver}$" "$file"; then
		echo "notice: ${file} already has a '## ${ver}' section — skipping insert"
		return
	fi
	local tmp_file
	tmp_file="$(mktemp)"
	awk -v ver="$ver" -v heading="$heading" '
		$0 == heading { print; print ""; print "## " ver; print ""; \
			print "- **Bug Fixes and Stability Improvements**"; \
			print "  - TODO"; next }
		{ print }
	' "$file" > "$tmp_file"
	mv "$tmp_file" "$file"
	track_change "$file"
	echo "notice: inserted placeholder '## ${ver}' section into ${file} — fill in before merging"
}

CHART_YAML="helm-charts/Chart.yaml"
MAKEFILE="Makefile"
CONF_PY="docs/conf.py"
INDEX_MD="docs/index.md"
HELM_MD="docs/installation/kubernetes-helm.md"
EXAMPLE_YAML="example/deviceConfigs_example.yaml"
RELEASENOTES_MD="docs/releasenotes.md"

for f in "$CHART_YAML" "$MAKEFILE" "$CONF_PY" "$INDEX_MD" \
	"$HELM_MD" "$EXAMPLE_YAML" "$RELEASENOTES_MD"; do
	require_file "$f"
done

# helm-charts/Chart.yaml: version: / appVersion: (tag form)
sed -i -e "s|^version:.*|version: ${VERSION_TAG}|" \
	-e "s|^appVersion:.*|appVersion: \"${VERSION_TAG}\"|" \
	"$CHART_YAML"
track_change "$CHART_YAML"

# Makefile: PROJECT_VERSION (bare, quoted) + helm-install chart tarball (tag)
sed -i -e "s|^PROJECT_VERSION ?= .*|PROJECT_VERSION ?= \"${VERSION_NUM}\"|" \
	-e "s|device-config-manager-charts-v[0-9]\+\.[0-9]\+\.[0-9]\+\.tgz|device-config-manager-charts-${VERSION_TAG}.tgz|g" \
	"$MAKEFILE"
track_change "$MAKEFILE"

# docs/conf.py: version = "..." (bare)
sed -i "s|^version = .*|version = \"${VERSION_NUM}\"|" "$CONF_PY"
track_change "$CONF_PY"

# docs/index.md: image tag (tag)
sed -i "s|rocm/device-config-manager:v[0-9]\+\.[0-9]\+\.[0-9]\+|rocm/device-config-manager:${VERSION_TAG}|g" \
	"$INDEX_MD"
track_change "$INDEX_MD"

# docs/installation/kubernetes-helm.md: helm tag + chart tarball (tag)
sed -i -e "s|tag: v[0-9]\+\.[0-9]\+\.[0-9]\+|tag: ${VERSION_TAG}|g" \
	-e "s|device-config-manager-charts-v[0-9]\+\.[0-9]\+\.[0-9]\+\.tgz|device-config-manager-charts-${VERSION_TAG}.tgz|g" \
	"$HELM_MD"
track_change "$HELM_MD"

# example/deviceConfigs_example.yaml: image tag (tag)
sed -i "s|rocm/device-config-manager:v[0-9]\+\.[0-9]\+\.[0-9]\+|rocm/device-config-manager:${VERSION_TAG}|g" \
	"$EXAMPLE_YAML"
track_change "$EXAMPLE_YAML"

# docs/releasenotes.md: conditional placeholder insert (tag)
insert_releasenotes_placeholder "$RELEASENOTES_MD" "# Release Notes" "$VERSION_TAG"

echo ""
echo "Bumped version to ${VERSION_TAG} in:"
printf '  %s\n' "${CHANGED_FILES[@]}" | sort -u
