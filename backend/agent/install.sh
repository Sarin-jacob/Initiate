#!/bin/bash
set -e

echo "Installing EdgeAuth Agent..."

# 1. Create directories
sudo mkdir -p /opt/edgeauth
sudo mkdir -p /etc/edgeauth

# 2. Move binary and config (assuming you built it as 'edgeauth-agent')
sudo cp edgeauth-agent /opt/edgeauth/
sudo cp config.yml /opt/edgeauth/
sudo chmod +x /opt/edgeauth/edgeauth-agent

# 3. Generate key if it doesn't exist
if [ ! -f /etc/edgeauth/agent.key ]; then
    echo "Generating new Ed25519 key..."
    # Note: Requires your keygen logic compiled to a binary, or just run it via go run if Go is installed
    cd /opt/edgeauth && go run /path/to/your/keygen.go
    sudo mv agent.key /etc/edgeauth/
fi

# 4. Create Systemd Service File
cat <<EOF | sudo tee /etc/systemd/system/edgeauth-agent.service
[Unit]
Description=EdgeAuth Target Provisioning Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/opt/edgeauth/edgeauth-agent --config /opt/edgeauth/config.yml
Restart=always
RestartSec=5
User=root
# Hardening options (optional but recommended)
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable edgeauth-agent
sudo systemctl restart edgeauth-agent

echo "Installation complete. Service is running."
echo "View logs with: sudo journalctl -fu edgeauth-agent"