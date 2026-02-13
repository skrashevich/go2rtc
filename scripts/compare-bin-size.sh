#!/bin/bash
# Compare binary size between current branch and master
# Usage: ./scripts/compare-bin-size.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get current branch name
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
ORIGINAL_BRANCH=$CURRENT_BRANCH

# Temporary directory for binaries
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

MASTER_BIN="$TMP_DIR/go2rtc_master"
CURRENT_BIN="$TMP_DIR/go2rtc_current"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Binary Size Comparison Tool${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Current branch: ${YELLOW}${CURRENT_BRANCH}${NC}"
echo ""

# Save current changes if any
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo -e "${YELLOW}Warning: You have uncommitted changes${NC}"
    echo -e "${YELLOW}Stashing changes before comparison...${NC}"
    git stash push -m "temp-stash-for-size-comparison" > /dev/null 2>&1
    STASHED=true
fi

# Build master branch
echo -e "${BLUE}Building master branch...${NC}"
git checkout master --quiet 2>/dev/null
go build -ldflags "-s -w" -trimpath -o "$MASTER_BIN" . 2>&1 | grep -i "error" && {
    echo -e "${RED}Build failed on master!${NC}"
    git checkout "$ORIGINAL_BRANCH" --quiet
    exit 1
}
MASTER_SIZE=$(wc -c < "$MASTER_BIN")
MASTER_SIZE_MB=$(echo "scale=2; $MASTER_SIZE / 1048576" | bc)

# Build current branch
echo -e "${BLUE}Building ${ORIGINAL_BRANCH} branch...${NC}"
git checkout "$ORIGINAL_BRANCH" --quiet 2>/dev/null
go build -ldflags "-s -w" -trimpath -o "$CURRENT_BIN" . 2>&1 | grep -i "error" && {
    echo -e "${RED}Build failed on ${ORIGINAL_BRANCH}!${NC}"
    exit 1
}
CURRENT_SIZE=$(wc -c < "$CURRENT_BIN")
CURRENT_SIZE_MB=$(echo "scale=2; $CURRENT_SIZE / 1048576" | bc)

# Restore stashed changes if any
if [ "$STASHED" = true ]; then
    echo -e "${BLUE}Restoring stashed changes...${NC}"
    git stash pop --quiet 2>/dev/null
fi

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
printf "  %-20s  %12s  %10s\n" "master" "$(printf "%'d" $MASTER_SIZE)" "${MASTER_SIZE_MB} MB"
printf "  %-20s  %12s  %10s\n" "$ORIGINAL_BRANCH" "$(printf "%'d" $CURRENT_SIZE)" "${CURRENT_SIZE_MB} MB"
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
