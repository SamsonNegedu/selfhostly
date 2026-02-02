# =============================================================================
# Selfhostly Frontend - Static Assets Only
# =============================================================================
# This serves the web UI via Node.js static server:
#   - HTML, CSS, JavaScript bundles
#   - Static assets (images, fonts, icons)
#   - SPA routing (all routes serve index.html)
#
# API calls use relative URLs (same origin) - Cloudflare routes them to gateway
# No backend logic, no database access, no authentication
# Pure static file server
# =============================================================================

# Stage 1: Build Frontend
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --silent

# Copy source and build
COPY . ./

# Build for production
RUN npm run build

# Stage 2: Production - Serve static files
FROM node:20-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001

# Copy built files from builder
COPY --from=builder /app/dist ./dist

# Change ownership
RUN chown -R nodejs:nodejs /app

# Switch to non-root user
USER nodejs

EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/ || exit 1

# Use npx serve to serve static files
# -s flag enables SPA mode (all routes serve index.html)
# -l 80 sets the port to 80
CMD ["npx", "-y", "serve", "-s", "dist", "-l", "80"]
