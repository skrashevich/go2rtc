#!/bin/bash
# Compare binary size between current branch and master
# Usage: ./scripts/compare-bin-size.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Keep initial repo state so it can always be restored on exit
ORIGINAL_HEAD=$(git rev-parse --verify HEAD)
if ORIGINAL_BRANCH=$(git symbolic-ref --quiet --short HEAD 2>/dev/null); then
    ORIGINAL_REF="$ORIGINAL_BRANCH"
else
    ORIGINAL_BRANCH=""
    ORIGINAL_REF="$ORIGINAL_HEAD"
fi

# Temporary directory for binaries
TMP_DIR=$(mktemp -d)
STASH_REF=""
STASH_MESSAGE="compare-bin-size-$(date +%s)-$$"

cleanup() {
    local exit_code="$1"
    local cleanup_failed=false

    set +e

    git checkout "$ORIGINAL_REF" --quiet >/dev/null 2>&1 || cleanup_failed=true

    if [ -n "$STASH_REF" ]; then
        git stash pop --quiet "$STASH_REF" >/dev/null 2>&1 || {
            echo -e "${YELLOW}Warning: couldn't auto-restore stashed changes (${STASH_REF})${NC}" >&2
            cleanup_failed=true
        }
    fi

    rm -rf "$TMP_DIR"

    if [ "$cleanup_failed" = true ] && [ "$exit_code" -eq 0 ]; then
        exit_code=1
    fi

    trap - EXIT
    exit "$exit_code"
}
trap 'cleanup $?' EXIT

MASTER_BIN="$TMP_DIR/go2rtc_master"
CURRENT_BIN="$TMP_DIR/go2rtc_current"
BASE_BRANCH="master"
CURRENT_BRANCH="${ORIGINAL_BRANCH:-detached@${ORIGINAL_HEAD:0:12}}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Binary Size Comparison Tool${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Current branch: ${YELLOW}${CURRENT_BRANCH}${NC}"
echo ""

# Save current changes if any (including untracked files)
if [ -n "$(git status --porcelain --untracked-files=all)" ]; then
    echo -e "${YELLOW}Warning: You have uncommitted changes${NC}"
    echo -e "${YELLOW}Stashing changes before comparison...${NC}"
    git stash push --include-untracked --quiet -m "$STASH_MESSAGE"
    STASH_REF="stash@{0}"
fi

# Build master branch
echo -e "${BLUE}Building ${BASE_BRANCH} branch...${NC}"
git checkout "$BASE_BRANCH" --quiet
if ! go build -ldflags "-s -w" -trimpath -o "$MASTER_BIN" .; then
    echo -e "${RED}Build failed on ${BASE_BRANCH}!${NC}"
    exit 1
fi
MASTER_SIZE=$(wc -c < "$MASTER_BIN")
MASTER_SIZE_MB=$(echo "scale=2; $MASTER_SIZE / 1048576" | bc)

# Build current branch
echo -e "${BLUE}Building ${CURRENT_BRANCH} branch...${NC}"
git checkout "$ORIGINAL_REF" --quiet
if ! go build -ldflags "-s -w" -trimpath -o "$CURRENT_BIN" .; then
    echo -e "${RED}Build failed on ${CURRENT_BRANCH}!${NC}"
    exit 1
fi
CURRENT_SIZE=$(wc -c < "$CURRENT_BIN")
CURRENT_SIZE_MB=$(echo "scale=2; $CURRENT_SIZE / 1048576" | bc)

# Calculate difference
DIFF=$((CURRENT_SIZE - MASTER_SIZE))
DIFF_ABS=${DIFF#-}  # Absolute value
DIFF_KB=$(echo "scale=1; $DIFF / 1024" | bc)
DIFF_MB=$(echo "scale=2; $DIFF / 1048576" | bc | sed 's/^\./0./')
PERCENT=$(echo "scale=2; $DIFF * 100 / $MASTER_SIZE" | bc)

# Output results
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Results${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
printf "  %-20s  %12s  %10s\n" "Branch" "Size" "Size (MB)"
echo "  ---------------------------------------------------"
printf "  %-20s  %12s  %10s\n" "$BASE_BRANCH" "$(printf "%'d" $MASTER_SIZE)" "${MASTER_SIZE_MB} MB"
printf "  %-20s  %12s  %10s\n" "$CURRENT_BRANCH" "$(printf "%'d" $CURRENT_SIZE)" "${CURRENT_SIZE_MB} MB"
echo ""
echo "  ---------------------------------------------------"

# Format numbers with thousands separator
format_number() {
    printf "%'d" $1
}

# Color coding for difference
if [ $DIFF -gt 0 ]; then
    echo -e "  ${RED}Difference:           +${DIFF_KB} KB (+${PERCENT}%)${NC}"
    echo -e "  ${RED}                     (≈ +${DIFF_MB} MB)${NC}"
elif [ $DIFF -lt 0 ]; then
    echo -e "  ${GREEN}Difference:           ${DIFF_KB} KB (${PERCENT}%)${NC}"
    echo -e "  ${GREEN}                     (≈ ${DIFF_MB} MB)${NC}"
else
    echo -e "  ${GREEN}Difference:           no change${NC}"
fi
echo ""

# Warn if growth is significant
if [ $DIFF -gt 1048576 ]; then  # More than 1MB
    echo -e "${YELLOW}⚠️  Warning: Binary size increased by more than 1MB!${NC}"
elif [ $DIFF -gt 524288 ]; then  # More than 512KB
    echo -e "${YELLOW}⚠️  Note: Binary size increased by more than 512KB${NC}"
elif [ $DIFF -gt 0 ]; then
    echo -e "${GREEN}✓ Binary size change is minimal${NC}"
fi
echo ""
