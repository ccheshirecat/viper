FROM chromedp/headless-shell:latest

# Install required packages for initramfs creation and networking
USER root
RUN apt-get update && apt-get install -y \
    kmod \
    initramfs-tools \
    busybox-static \
    iproute2 \
    iptables \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy viper-agent binary (expected to be built before Docker build)
COPY bin/viper-agent /usr/local/bin/viper-agent
RUN chmod +x /usr/local/bin/viper-agent

# Configure agent as the init process
RUN echo '#!/bin/sh\nexec /usr/local/bin/viper-agent "$@"' > /init && chmod +x /init

# Create required directories
RUN mkdir -p /var/viper/tasks /tmp/viper

# Configure minimal network setup for VM environment
RUN echo '#!/bin/sh\n\
# Basic network setup for VM\n\
ip link set lo up\n\
if [ -n "$VM_IP" ]; then\n\
    ip addr add $VM_IP dev eth0\n\
    ip link set eth0 up\n\
    if [ -n "$VM_GATEWAY" ]; then\n\
        ip route add default via $VM_GATEWAY\n\
    fi\n\
else\n\
    # Try DHCP fallback\n\
    dhclient eth0 2>/dev/null || true\n\
fi\n\
' > /usr/local/bin/setup-network.sh && chmod +x /usr/local/bin/setup-network.sh

# Ensure headless-shell starts properly in VM environment
ENV DISPLAY=:99
ENV CHROME_FLAGS="--no-sandbox --disable-gpu --remote-debugging-port=9222"

# VM will boot directly to our agent
CMD ["/usr/local/bin/viper-agent", "--listen=:8080"]