# data-backup

A lightweight SQLite backup service for Linux. Backs up configured databases to Google Drive via rclone on a configurable schedule, with local staging copies and change detection via SHA256 checksums.

## Features

- SHA-256-based — only backs up when the database has a new hash
- Safe backups via `sqlite3 .backup` — will not corrupt a live database
- Remote sync via rclone
- Cross-references local registry against remote storage
- Periodic reporting via systemd journal

## Dependencies

- `sqlite3`
- `rclone` (configured with a Google Drive remote*)
- Go 1.24+ (for building)

## Build

```bash
go build -o data-backup
sudo mv data-backup /usr/local/bin/
```

## Directory Setup

```bash
sudo mkdir -p /etc/data-backup
sudo mkdir -p /var/lib/data-backup/your_db_name
sudo chown -R $USER:$USER /var/lib/data-backup
```

## Configuration

Copy the example config and edit it:

```bash
sudo cp config_example.toml /etc/data-backup/config.toml
```

See `config_example.toml` for all available options.

## rclone Setup

Configure a Google Drive remote named to match your `remote_path`:

```bash
rclone config
```

Follow the OAuth flow. On a headless machine, run `rclone config` on a desktop with a browser and copy `~/.config/rclone/rclone.conf` to the server.

## Systemd Installation

```bash
sudo cp data-backup.service data-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now data-backup.timer
```

Test a manual run:

```bash
sudo systemctl start data-backup.service
journalctl -u data-backup.service -f
```

## Logs

All output goes to the systemd journal:

```bash
journalctl -u data-backup.service
```
