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

# 3. Gather Arguments (with Readline support for arrows/pasting)
echo -e "\n${YELLOW}>>> Configuration...${NC}"

# Helper function to handle TTY input safely during a curl|bash pipe
read_input() {
    local prompt=$1
    local var_name=$2
    local silent=$3

    if [ "$silent" == "silent" ]; then
        # Silent input for passwords, no readline needed here
        read -r -s -p "$prompt" value </dev/tty
        echo "" </dev/tty # Print newline after hidden input
    else
        # -e enables Readline (arrows, pasting), -r prevents backslash escaping
        read -r -e -p "$prompt" value </dev/tty
    fi
    eval "$var_name=\"\$value\""
}

read_input "Colonies Host (e.g., colony.colonypm.xyz): " HOST
read_input "Port (e.g., 443): " PORT
read_input "Use TLS / Secure Connection? (y/n): " USE_TLS
read_input "Executor Name (e.g., worker-01): " EXE_NAME
read_input "Colony Private Key: " PRVKEY "silent"

# Handle the -insecure flag based on the TLS question
INSECURE="true"
if [[ "$USE_TLS" =~ ^[Yy]$ ]]; then
    INSECURE="false"
fi

# 4. Start the Executor
echo -e "\n${GREEN}==========================================${NC}"
echo -e "         LAUNCHING EXECUTOR               "
echo -e "${GREEN}==========================================${NC}\n"

# Run the Go binary directly with the provided flags
./spawn-executor \
    -host "$HOST" \
    -port "$PORT" \
    -insecure=$INSECURE \
    -key "$PRVKEY" \
    -name "$EXE_NAME"
