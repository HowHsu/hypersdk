#!/usr/bin/env bash
# Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
# See the file LICENSE for licensing terms.

set -euo pipefail

if ! [[ "$0" =~ scripts/lint.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)
# shellcheck source=/scripts/common/utils.sh
source "${REPO_ROOT}"/scripts/common/utils.sh

if [ "$#" -eq 0 ]; then
  # by default, check all source code
  # to test only "mempool" package
  # ./scripts/lint.sh ./mempool/...
  TARGET="./..."
else
  TARGET="${1}"
fi

add_license_headers -check

# by default, "./scripts/lint.sh" runs all lint tests
TESTS=${TESTS:-"golangci_lint gci"}

# https://github.com/golangci/golangci-lint/releases
function test_golangci_lint {
  "${REPO_ROOT}"/bin/golangci-lint run --config .golangci.yml
}

function test_gci {
  # Ensure gci is installed first to ensure installation output isn't confused for non-compliant files
  "${REPO_ROOT}"/bin/gci --version

  FILES=$("${REPO_ROOT}"/bin/gci list --skip-generated -s standard -s default -s blank -s dot -s "prefix(github.com/ava-labs/hypersdk)" -s alias --custom-order .)
  if [[ "${FILES}" ]]; then
    echo ""
    echo "Some files need to be gci-ed:"
    echo "${FILES}"
    echo ""
    return 1
  fi
}

function run {
  local test="${1}"
  shift 1
  echo "START: '${test}' at $(date)"
  if "test_${test}" "$@" ; then
    echo "SUCCESS: '${test}' completed at $(date)"
  else
    echo "FAIL: '${test}' failed at $(date)"
    exit 255
  fi
}

echo "Running '$TESTS' at: $(date)"
for test in $TESTS; do
  run "${test}" "${TARGET}"
done

echo "ALL SUCCESS!"
