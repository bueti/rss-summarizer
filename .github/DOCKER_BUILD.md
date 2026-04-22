# Docker Build Workflow

This document explains how the Docker build and deployment process works.

## Overview

The GitHub Actions workflow automatically builds and pushes Docker images to GitHub Container Registry (GHCR) when you push to the `main` branch or create version tags.

## Build Process

### Backend (`backend/Dockerfile`)
- Multi-stage build using Go 1.26
- Compiles the API binary
- Final image: Alpine-based (~30MB)
- Includes database migrations

### Frontend (`frontend/Dockerfile`)
- Multi-stage build using Node 20
- **Important**: Uses pre-generated API types (committed to repo)
- Builds SvelteKit app for production
- Final image: Node Alpine-based (~150MB)

## Why Commit Generated API Types?

The frontend build needs TypeScript types generated from the backend's OpenAPI schema. During Docker build, the backend isn't running, so we can't fetch the schema. Therefore:

1. **Development**: `npm run dev` regenerates types from running backend
2. **Docker Build**: `npm run build:docker` uses committed types from `src/lib/api/generated.ts`

### Updating API Types

When you change the backend API:

```bash
# 1. Start backend
cd backend
docker compose up -d

# 2. Regenerate frontend types
cd ../frontend
npm run generate:api

# 3. Commit the updated types
git add src/lib/api/generated.ts
git commit -m "Update API types"
git push
```

The GitHub Actions workflow will rebuild both images with the updated types.

## Image Tags

Images are automatically tagged with:

- `latest` - Latest build from `main` branch
- `main-<sha>` - Specific commit SHA (e.g., `main-abc1234`)
- `v1.0.0` - Semantic version tags (create with `git tag v1.0.0`)
- `1.0` - Major.minor version (auto-derived from semver tags)

## Image Locations

After the workflow runs, images are available at:

```
ghcr.io/YOUR_USERNAME/rss-summarizer/backend:latest
ghcr.io/YOUR_USERNAME/rss-summarizer/frontend:latest
```

These are private by default and require authentication to pull.

## Triggering Builds

### Automatic (Recommended)
```bash
git add .
git commit -m "Your changes"
git push origin main
# Workflow runs automatically
```

### Tagged Release
```bash
git tag v1.0.0
git push origin v1.0.0
# Creates versioned images
```

### Manual Trigger
You can also trigger builds manually from the GitHub Actions tab in your repository.

## Build Optimization

The workflow uses GitHub Actions cache to speed up builds:
- Docker layer caching for faster rebuilds
- Only changed layers are rebuilt
- Typical rebuild time: 2-3 minutes

## Troubleshooting

### "Failed to generate API types"
- Make sure `frontend/src/lib/api/generated.ts` is committed
- Run `npm run generate:api` locally and commit the file

### "Failed to push to GHCR"
- Check that "Read and write permissions" are enabled in repo settings
- Go to Settings → Actions → General → Workflow permissions

### Build failures
- Check the Actions tab in GitHub for detailed logs
- Most common issue: missing generated.ts file

## Local Testing

Test the Docker build locally before pushing:

```bash
# Backend
docker build -t rss-backend ./backend

# Frontend (requires generated.ts to exist)
docker build -t rss-frontend ./frontend

# Test locally
docker run -p 8080:8080 rss-backend
docker run -p 3000:3000 rss-frontend
```

## Deployment

See [DEPLOYMENT.md](../DEPLOYMENT.md) for deployment instructions using the built images.
