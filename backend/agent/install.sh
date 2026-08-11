#!/bin/bash
set -e

echo "Installing NexusAuth Agent..."

# 1. Create directories
sudo mkdir -p /opt/nexusauth
sudo mkdir -p /etc/nexusauth

# 2. Move binary and config (assuming you built it as 'nexusauth-agent')
sudo cp nexusauth-agent /opt/nexusauth/
sudo cp config.yml /opt/nexusauth/
sudo chmod +x /opt/nexusauth/nexusauth-agent

# 3. Generate key if it doesn't exist
if [ ! -f /etc/nexusauth/agent.key ]; then
    echo "Generating new Ed25519 key..."
    # Note: Requires your keygen logic compiled to a binary, or just run it via go run if Go is installed
    cd /opt/nexusauth && go run /path/to/your/keygen.go
    sudo mv agent.key /etc/nexusauth/
fi

# 4. Create Systemd Service File
cat <<EOF | sudo tee /etc/systemd/system/nexusauth-agent.service
[Unit]
Description=NexusAuth Target Provisioning Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/nexusauth
ExecStart=/opt/nexusauth/nexusauth-agent --config /opt/nexusauth/config.yml
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
sudo systemctl enable nexusauth-agent
sudo systemctl restart nexusauth-agent

echo "Installation complete. Service is running."
echo "View logs with: sudo journalctl -fu nexusauth-agent"