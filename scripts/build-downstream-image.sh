#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=${MSGVAULT_DOWNSTREAM_VERSION:-v0.19.3-x1pher.2}
image=${MSGVAULT_IMAGE:-ghcr.io/x1pher/msgvault:0.19.3-x1pher.2}

if [[ "$version" != "v0.19.3-x1pher.2" ]]; then
  echo "unexpected downstream version: $version" >&2
  exit 1
fi

if commit=$(git rev-parse --verify HEAD 2>/dev/null); then
  :
else
  commit="pre-release-non-git"
fi

build_date=${MSGVAULT_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}

docker build \
  --build-arg "VERSION=$version" \
  --build-arg "COMMIT=$commit" \
  --build-arg "BUILD_DATE=$build_date" \
  --tag "$image" \
  .

docker run --rm --network none "$image" import-discord-observations --help >/dev/null
docker run --rm --network none "$image" mcp --help | grep -- '--read-only' >/dev/null
docker run --rm --network none "$image" version | grep "^msgvault $version$" >/dev/null

echo "built and contract-checked $image"
