#!/bin/bash

# Listmonk Database Backup Script
# Run this script regularly (daily/weekly) via cron or GitHub Actions

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="./backups"
BACKUP_FILE="listmonk_backup_${DATE}.sql"

# Create backup directory
mkdir -p $BACKUP_DIR

# Get database connection details from Fly.io
echo "Creating database backup..."

# Connect to Fly.io and create backup
flyctl postgres connect --app listmonk-db-1758978372 --database listmonk_app_1759010454 --command "pg_dump listmonk_app_1759010454" > "${BACKUP_DIR}/${BACKUP_FILE}"

# Compress the backup
gzip "${BACKUP_DIR}/${BACKUP_FILE}"

echo "Backup created: ${BACKUP_DIR}/${BACKUP_FILE}.gz"

# Optional: Upload to cloud storage (AWS S3, Google Cloud, etc.)
# aws s3 cp "${BACKUP_DIR}/${BACKUP_FILE}.gz" s3://your-backup-bucket/listmonk/

# Clean up old backups (keep last 30 days)
find $BACKUP_DIR -name "listmonk_backup_*.sql.gz" -mtime +30 -delete

echo "Backup completed successfully!"
