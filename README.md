# GoSSHTerm

A web-based terminal that connects to an SSH server via xterm.js. It runs on macOS and Linux (tested on Ubuntu).

GoSSHTerm requires an SSH server that accepts password authentication.

## Warning: Curiosity-Driven Code Ahead

This is a "can I do it?" project, not a "should I do it?" project. It exists to prove a point, explore an idea, and generally satisfy personal curiosity.

It is **not production-ready**. If you deploy it anyway, please do so knowingly, cheerfully, and with a strong appreciation for _consequences_.

## Building

```bash
go build -o gosshterm .
```

Then open http://localhost:8000 in your browser, enter your SSH credentials, and connect.

## Usage

```
Usage of ./gosshterm:
  -http string
        HTTP server port (default "8000")
  -ssh-host string
        SSH host (default "localhost")
  -ssh-port string
        SSH port (default "22")
```

### Examples

```bash
# Default: localhost:2222, HTTP on port 8000
./gosshterm

# Custom HTTP port
./gosshterm -http 9000

# Connect to a remote SSH server
./gosshterm -ssh-host 192.168.1.100 -ssh-port 22
```

## Running as a _Linux_ systemd service

Create `/etc/systemd/system/gosshterm.service`:

```ini
[Unit]
Description=GoSSHTerm Web Terminal
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
WorkingDirectory=/opt/gosshterm
ExecStart=/opt/gosshterm/gosshterm -ssh-port 22
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo mkdir /opt/gosshterm/
sudo cp gosshterm /opt/gosshterm/
sudo cp -r static /opt/gosshterm/
sudo systemctl daemon-reload
sudo systemctl enable gosshterm
sudo systemctl start gosshterm
```

## Security Notes

- Credentials are sent via WebSocket query parameters over the connection
- Use HTTPS in production (put behind a reverse proxy like nginx/caddy)
- The SSH host/port is controlled server-side; users only provide username/password
- Consider restricting the SSH server to localhost (`ListenAddress 127.0.0.1`)
