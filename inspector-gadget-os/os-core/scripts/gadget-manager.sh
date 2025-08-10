#!/bin/bash
# Inspector Gadget Manager - Core gadget lifecycle management

GADGET_DIR="/opt/inspector-gadget/gadgets"
CONFIG_DIR="/etc/inspector-gadget"
DATA_DIR="/var/lib/inspector-gadget"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function show_help() {
    cat << EOF
Inspector Gadget Manager - "Go Go Gadget!"

Usage: gadget-manager [command] [options]

Commands:
    install <gadget>    Install a new gadget
    remove <gadget>     Remove an installed gadget
    list               List all installed gadgets
    start <gadget>     Start a gadget service
    stop <gadget>      Stop a gadget service
    status <gadget>    Show gadget status
    update <gadget>    Update a gadget to latest version

Examples:
    gadget-manager install network-scanner
    gadget-manager start ultron-assistant
    gadget-manager list

EOF
}

function list_gadgets() {
    echo -e "${GREEN}Installed Gadgets:${NC}"
    if [ -d "$GADGET_DIR" ]; then
        for gadget in "$GADGET_DIR"/*; do
            if [ -d "$gadget" ]; then
                name=$(basename "$gadget")
                if [ -f "$gadget/gadget.yaml" ]; then
                    version=$(grep "version:" "$gadget/gadget.yaml" | awk '{print $2}')
                    echo "  - $name (v$version)"
                else
                    echo "  - $name"
                fi
            fi
        done
    else
        echo "  No gadgets installed"
    fi
}

function install_gadget() {
    gadget_name=$1
    echo -e "${YELLOW}Installing gadget: $gadget_name${NC}"
    
    # Create gadget directory
    mkdir -p "$GADGET_DIR/$gadget_name"
    
    # TODO: Download and install gadget from registry
    echo -e "${GREEN}✓ Gadget '$gadget_name' installed successfully${NC}"
}

function start_gadget() {
    gadget_name=$1
    echo -e "${YELLOW}Starting gadget: $gadget_name${NC}"
    
    # TODO: Start gadget service/container
    echo -e "${GREEN}✓ Gadget '$gadget_name' started${NC}"
}

# Main command processing
case "$1" in
    install)
        install_gadget "$2"
        ;;
    list)
        list_gadgets
        ;;
    start)
        start_gadget "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Go Go Gadget... what?"
        show_help
        exit 1
        ;;
esac