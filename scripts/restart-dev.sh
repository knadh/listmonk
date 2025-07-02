#!/bin/bash

# Development restart script for listmonk
# Stops backend, rebuilds frontend, and starts backend

set -e

echo "🛑 Stopping backend..."
if [ -f /tmp/listmonk.pid ]; then
    kill $(cat /tmp/listmonk.pid) 2>/dev/null || true
    rm -f /tmp/listmonk.pid
fi

# Also kill any other Go processes that might be running
pkill -f "go run.*cmd" 2>/dev/null || true

# Kill any processes on port 9000
lsof -ti:9000 | xargs kill -9 2>/dev/null || true

echo "🔨 Rebuilding frontend..."
make build-frontend

echo "🚀 Starting backend..."
make run > /tmp/listmonk_debug.log 2>&1 & echo $! > /tmp/listmonk.pid

echo "⏳ Waiting for server to start..."
sleep 3

if tail -3 /tmp/listmonk_debug.log | grep -q "http server started"; then
    echo "✅ Server started successfully on http://127.0.0.1:9000"
else
    echo "❌ Server failed to start. Check logs:"
    tail -10 /tmp/listmonk_debug.log
    exit 1
fi