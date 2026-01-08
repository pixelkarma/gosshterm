# GoTerm

A web-based terminal that connects to an SSH server via xterm.js.

## Building

```bash
go build -o goterm .
```

## Installation

### 1. Set up an SSH server with password authentication

GoTerm requires an SSH server that accepts password authentication. If your system's SSH server doesn't allow password auth, you can run a separate instance:

```bash
# Create config directory
sudo mkdir -p /etc/ssh/goterm

# Create sshd config with password auth enabled
sudo tee /etc/ssh/goterm/sshd_config > /dev/null << 'EOF'
Port 2222
ListenAddress 127.0.0.1
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_rsa_key
PasswordAuthentication yes
PermitRootLogin no
PubkeyAuthentication yes
UsePAM yes
Subsystem sftp /usr/lib/openssh/sftp-server
EOF

# Ensure privilege separation directory exists
sudo mkdir -p /run/sshd

# Start sshd on port 2222
sudo /usr/sbin/sshd -f /etc/ssh/goterm/sshd_config
```

### 2. Create a user (optional)

```bash
sudo useradd -m -s /bin/bash myuser
echo "myuser:mypassword" | sudo chpasswd
```

### 3. Run GoTerm

```bash
./goterm
```

## Usage

```
Usage of ./goterm:
  -http string
        HTTP server port (default "8000")
  -ssh-host string
        SSH host (default "localhost")
  -ssh-port string
        SSH port (default "2222")
  -ssh-user string
        SSH username - used as fallback (default "david")
  -ssh-pass string
        SSH password - used as fallback (default "qwerty")
```

### Examples

```bash
# Default: localhost:2222, HTTP on port 8000
./goterm

# Custom HTTP port
./goterm -http 9000

# Connect to a remote SSH server
./goterm -ssh-host 192.168.1.100 -ssh-port 22

# Connect to standard SSH port on localhost
./goterm -ssh-port 22
```

Then open http://localhost:8000 in your browser, enter your SSH credentials, and connect.

## Running as a systemd service

Create `/etc/systemd/system/goterm.service`:

```ini
[Unit]
Description=GoTerm Web Terminal
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
WorkingDirectory=/opt/goterm
ExecStart=/opt/goterm/goterm -ssh-port 22
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo cp goterm /opt/goterm/
sudo cp -r static /opt/goterm/
sudo systemctl daemon-reload
sudo systemctl enable goterm
sudo systemctl start goterm
```

## Security Notes

- Credentials are sent via WebSocket query parameters over the connection
- Use HTTPS in production (put behind a reverse proxy like nginx/caddy)
- The SSH host/port is controlled server-side; users only provide username/password
- Consider restricting the SSH server to localhost (`ListenAddress 127.0.0.1`)
