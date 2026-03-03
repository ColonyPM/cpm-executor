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
    echo -e "${RED}${CROSS} Docker not found.${NC} Please install: https://docs.docker.com/get-docker/"; exit 1
else
    echo -e "  ${CHECK} Docker"
fi

# Python 3
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}${CROSS} Python 3 not found.${NC}"; exit 1
else
    echo -e "  ${CHECK} Python 3"
fi

# Pip 3
if ! command -v pip3 &> /dev/null; then
    echo -e "${RED}${CROSS} Pip3 not found.${NC} Installing...";
    sudo apt-get update -qq && sudo apt-get install -y python3-pip -qq > /dev/null || { echo -e "${RED}Failed to install pip3.${NC}"; exit 1; }
fi
echo -e "  ${CHECK} Pip 3"

# Pycolonies
echo -en "  Installing pycolonies package... "
pip3 install pycolonies --quiet --break-system-packages 2>/dev/null || pip3 install pycolonies --quiet
echo -e "\r  ${CHECK} Pycolonies installed.             "


# 2. Download the Executor Script
echo -e "\n${YELLOW}>>> Downloading Payload...${NC}"
RAW_URL="https://raw.githubusercontent.com/ColonyPM/cpm-executor/refs/heads/main/executor.py"

if curl -sL "$RAW_URL" -o executor.py; then
    chmod +x executor.py
    echo -e "  ${CHECK} executor.py downloaded."
else
    echo -e "  ${RED}${CROSS} Failed to download script.${NC}"; exit 1
fi


# 3. Gather Arguments
echo -e "\n${YELLOW}>>> Configuration...${NC}"
read -p "Colonies Host (e.g., colony.colonypm.xyz): " HOST
read -p "Port (e.g., 443): " PORT
read -p "Use TLS? (y/n): " USE_TLS
read -p "Executor Name: " EXE_NAME
read -sp "Colony Private Key: " PRVKEY
echo "" # Newline after hidden password input

TLS_FLAG=""
if [[ "$USE_TLS" =~ ^[Yy]$ ]]; then
    TLS_FLAG="--tls"
fi


# 4. Start the Executor
echo -e "\n${YELLOW}>>> Launching Executor...${NC}"

# Start normally (in foreground, no nohup)
python3 executor.py \
    --host "$HOST" \
    --port "$PORT" \
    --executor-name "$EXE_NAME" \
    --prvkey "$PRVKEY" \
    $TLS_FLAG
