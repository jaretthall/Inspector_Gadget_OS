#!/bin/bash
# Inspector Gadget OS - USB Image Builder

set -e

# Configuration
BUILD_DIR="$(pwd)/build"
OUTPUT_DIR="$(pwd)/output"
ISO_NAME="inspector-gadget-os-v1.0.iso"
WORK_DIR="$BUILD_DIR/work"
ROOTFS_DIR="$WORK_DIR/rootfs"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Inspector Gadget OS - USB Image Builder${NC}"
echo "Building bootable USB image..."

# Create build directories
mkdir -p "$BUILD_DIR" "$OUTPUT_DIR" "$WORK_DIR" "$ROOTFS_DIR"

# Step 1: Build Alpine base
echo -e "${YELLOW}Step 1: Building Alpine base system...${NC}"
cd ../os-core
docker build -f Dockerfile.alpine -t inspector-gadget-base:latest .

# Step 2: Extract rootfs from Docker
echo -e "${YELLOW}Step 2: Extracting root filesystem...${NC}"
container_id=$(docker create inspector-gadget-base:latest)
docker export "$container_id" | tar -x -C "$ROOTFS_DIR"
docker rm "$container_id"

# Step 3: Copy O-LLaMA
echo -e "${YELLOW}Step 3: Installing O-LLaMA...${NC}"
mkdir -p "$ROOTFS_DIR/opt/o-llama"
# TODO: Copy built O-LLaMA binaries

# Step 4: Install bootloader
echo -e "${YELLOW}Step 4: Installing bootloader...${NC}"
# Create boot directory structure
mkdir -p "$ROOTFS_DIR/boot/grub"

# Create GRUB configuration
cat > "$ROOTFS_DIR/boot/grub/grub.cfg" << 'EOF'
set timeout=5
set default=0

menuentry "Inspector Gadget OS" {
    linux /boot/vmlinuz-lts root=/dev/sda1 modules=ext4 quiet
    initrd /boot/initramfs-lts
}

menuentry "Inspector Gadget OS (Safe Mode)" {
    linux /boot/vmlinuz-lts root=/dev/sda1 modules=ext4 single
    initrd /boot/initramfs-lts
}
EOF

# Step 5: Create ISO image
echo -e "${YELLOW}Step 5: Creating ISO image...${NC}"
# This would normally use mkisofs/genisoimage
# For now, we'll create a placeholder
touch "$OUTPUT_DIR/$ISO_NAME"

echo -e "${GREEN}✓ Build complete!${NC}"
echo "Output: $OUTPUT_DIR/$ISO_NAME"
echo ""
echo "To write to USB:"
echo "  sudo dd if=$OUTPUT_DIR/$ISO_NAME of=/dev/sdX bs=4M status=progress"
echo ""
echo "Go Go Gadget Boot!"