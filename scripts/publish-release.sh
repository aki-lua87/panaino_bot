#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${GITHUB_SHA:?GITHUB_SHA is required}"
: "${GH_REPO:?GH_REPO is required}"
[[ "$GITHUB_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid commit SHA' >&2; exit 1; }
tag="build-$GITHUB_SHA"
(cd dist && sha256sum --check SHA256SUMS)

# Re-running a successful workflow must not replace an already published release.
if draft=$(gh release view "$tag" --json isDraft --jq .isDraft 2>/dev/null); then
    if [[ "$draft" == false ]]; then
        echo "Already published: $tag"
        exit 0
    fi
else
    if ref_sha=$(gh api "repos/$GH_REPO/git/ref/tags/$tag" --jq .object.sha 2>/dev/null); then
        [[ "$ref_sha" == "$GITHUB_SHA" ]] || { echo 'Release tag points to a different commit' >&2; exit 1; }
    else
        gh api --method POST "repos/$GH_REPO/git/refs" -f "ref=refs/tags/$tag" -f "sha=$GITHUB_SHA" >/dev/null
    fi
    printf 'Commit: `%s`\n\nStatic Linux amd64 binary for Ubuntu 16.04.\nDownload panaino-bot and verify SHA256SUMS before updating.\n' "$GITHUB_SHA" > dist/release-notes.md
    gh release create "$tag" --verify-tag --draft --title "Build ${GITHUB_SHA:0:12}" --notes-file dist/release-notes.md
fi

# Upload completely while still a draft: latest never exposes partial assets.
gh release upload "$tag" dist/panaino-bot dist/SHA256SUMS --clobber
head_sha=$(gh api "repos/$GH_REPO/git/ref/heads/master" --jq .object.sha)
latest=false
[[ "$head_sha" != "$GITHUB_SHA" ]] || latest=true
gh release edit "$tag" --draft=false "--latest=$latest"
echo "Published: $tag (latest=$latest)"
