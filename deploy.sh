#!/bin/bash
set -e

# RSS Summarizer Deployment Script for Hetzner/VPS
# This script sets up the application on a fresh server

echo "🚀 RSS Summarizer Deployment Script"
echo "===================================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Installing..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER
    echo "✅ Docker installed. Please log out and back in, then run this script again."
    exit 0
fi

# Check if Tailscale is installed
if ! command -v tailscale &> /dev/null; then
    echo "❌ Tailscale is not installed. Installing..."
    curl -fsSL https://tailscale.com/install.sh | sh
    echo "✅ Tailscale installed. Please run 'sudo tailscale up' to authenticate."
    exit 0
fi

# Check if logged into GHCR
if ! docker info 2>/dev/null | grep -q "ghcr.io"; then
    echo "❌ Not logged into GitHub Container Registry"
    echo ""
    echo "Please create a GitHub Personal Access Token:"
    echo "1. Go to: https://github.com/settings/tokens"
    echo "2. Generate new token (classic) with 'read:packages' scope"
    echo "3. Run: echo 'YOUR_TOKEN' | docker login ghcr.io -u YOUR_USERNAME --password-stdin"
    echo ""
    exit 1
fi

# Create directory
mkdir -p ~/rss-summarizer
cd ~/rss-summarizer

# Download necessary files
echo "📥 Downloading deployment files..."
if [ -f "docker-compose.prod.yml" ]; then
    echo "⚠️  docker-compose.prod.yml already exists, skipping download"
else
    echo "Downloading docker-compose.prod.yml..."
    # User should provide their own repo URL
    echo "⚠️  Please manually copy docker-compose.prod.yml and .env.prod.example to this directory"
fi

# Check if .env exists
if [ ! -f ".env" ]; then
    if [ -f ".env.prod.example" ]; then
        cp .env.prod.example .env
        echo "⚠️  Created .env from example. Please edit it with your settings:"
        echo "    nano .env"
        exit 0
    else
        echo "❌ .env.prod.example not found. Please copy it to this directory first."
        exit 1
    fi
fi

# Get Tailscale IP
TAILSCALE_IP=$(tailscale ip -4 2>/dev/null || echo "unknown")
echo "📡 Your Tailscale IP: $TAILSCALE_IP"
echo "   Set ORIGIN=http://$TAILSCALE_IP:3000 in .env"

# Pull images
echo ""
echo "📦 Pulling Docker images..."
docker compose -f docker-compose.prod.yml pull

# Start services
echo ""
echo "🚀 Starting services..."
docker compose -f docker-compose.prod.yml up -d

# Wait for services to be healthy
echo ""
echo "⏳ Waiting for services to start..."
sleep 10

# Show status
echo ""
echo "📊 Service Status:"
docker compose -f docker-compose.prod.yml ps

echo ""
echo "✅ Deployment complete!"
echo ""
echo "🌐 Access the application at: http://$TAILSCALE_IP:3000"
echo ""
echo "📝 Useful commands:"
echo "  - View logs: docker compose -f docker-compose.prod.yml logs -f"
echo "  - Restart: docker compose -f docker-compose.prod.yml restart"
echo "  - Stop: docker compose -f docker-compose.prod.yml down"
echo "  - Update: docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d"
