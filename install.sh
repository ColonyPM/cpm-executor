#!/bin/bash
set -e

# Visual formatting
CHECK="\033[0;32m\xE2\x9C\x94\033[0m"
CROSS="\033[0;31m\xE2\x9C\x98\033[0m"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}==========================================${NC}"
echo -e "${GREEN}      Colonies Executor Setup             ${NC}"
echo -e "${CYAN}==========================================${NC}"

# 1. Dependency Checks
echo -e "\n${YELLOW}>>> Checking System Requirements...${NC}"

# Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}${CROSS} Docker not found.${NC} Please install Docker first."; exit 1
else
    echo -e "  ${CHECK} Docker"
fi

# 2. Download the Go Binary
echo -e "\n${YELLOW}>>> Downloading Payload...${NC}"
BIN_URL="https://github.com/ColonyPM/cpm-executor/releases/download/latest/spawn-executor"

if curl -sL "$BIN_URL" -o spawn-executor; then
    chmod +x spawn-executor
    echo -e "  ${CHECK} spawn-executor downloaded."
else
    echo -e "  ${RED}${CROSS} Failed to download binary.${NC}"; exit 1
fi

# 3. Configuration from Environment Variables
echo -e "\n${YELLOW}>>> Checking Configuration...${NC}"

# Ensure no defaults are used and all variables are present
MISSING_VARS=false
for var in HOST PORT INSECURE EXE_NAME PRVKEY; do
    if [ -z "${!var}" ]; then
        echo -e "  ${RED}${CROSS} Missing required environment variable: $var${NC}"
        MISSING_VARS=true
    fi
done

if [ "$MISSING_VARS" = true ]; then
    echo -e "\n${RED}Error: Execution aborted. Please provide all required environment variables.${NC}"
    echo "Example: HOST=colony.colonypm.xyz PORT=443 INSECURE=false EXE_NAME=worker-01 PRVKEY=your_key ./setup.sh"
    exit 1
fi

# Print the values being used
echo -e "  ${CHECK} Host: $HOST"
echo -e "  ${CHECK} Port: $PORT"
echo -e "  ${CHECK} Insecure: $INSECURE"
echo -e "  ${CHECK} Name: $EXE_NAME"
echo -e "  ${CHECK} Key: ********"

# 4. Start the Executor
echo -e "\n${GREEN}==========================================${NC}"
echo -e "         LAUNCHING EXECUTOR               "
echo -e "${GREEN}==========================================${NC}\n"

# Run the Go binary directly with the provided flags
./spawn-executor \
    -host "$HOST" \
    -port "$PORT" \
    -insecure="$INSECURE" \
    -key "$PRVKEY" \
    -name "$EXE_NAME"
