# Frontend Dockerfile for SvelteKit (adapter-node)
# 1) Build stage
FROM node:20-alpine AS builder
WORKDIR /app

# Install dependencies
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install --no-audit --prefer-offline; fi

# Build the app
COPY . .
RUN npm run build

# 2) Runtime stage
FROM node:20-alpine AS runner
ENV NODE_ENV=production
WORKDIR /app

# Copy the built app
COPY --from=builder /app/build ./build

# Install runtime dependencies required by the build output
WORKDIR /app/build
RUN npm install --omit=dev --no-audit --prefer-offline

# Expose the default port used by SvelteKit adapter-node
EXPOSE 3000

# Start the server
CMD ["node", "/app/build"]
