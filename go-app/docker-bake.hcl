# Docker Bake configuration for multi-architecture builds
# Supports AMD64 and ARM64 platforms

variable "TAG" {
  default = "latest"
}

variable "REGISTRY" {
  default = "alert-history"
}

variable "REPO" {
  default = "localhost"
}

group "default" {
  targets = ["app"]
}

target "app" {
  name = "${REGISTRY}-${formatdate("YYYYMMDDhhmmss", timestamp())}"

  # Multi-platform support
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]

  # Build arguments
  args = {
    BUILDKIT_INLINE_CACHE = "1"
  }

  # Tags
  tags = [
    "${REPO}/${REGISTRY}:${TAG}",
    "${REPO}/${REGISTRY}:${formatdate("YYYY-MM-DD", timestamp())}",
    "${REPO}/${REGISTRY}:${substr(sha1(formatdate("YYYY-MM-DD-hhmmss", timestamp())), 0, 8)}"
  ]

  # Build context
  context = "."

  # Dockerfile
  dockerfile = "Dockerfile"

  # Build cache
  cache-from = [
    "type=registry,ref=${REPO}/${REGISTRY}:cache"
  ]

  cache-to = [
    "type=registry,ref=${REPO}/${REGISTRY}:cache,mode=max"
  ]

  # Push to registry
  output = ["type=registry,push=true"]
}

# Development target (single platform, faster builds)
target "dev" {
  inherits = ["app"]

  platforms = ["linux/amd64"]

  tags = [
    "${REPO}/${REGISTRY}:dev",
    "${REPO}/${REGISTRY}:dev-${formatdate("YYYY-MM-DD", timestamp())}"
  ]

  cache-from = [
    "type=local,src=/tmp/.buildx-cache"
  ]

  cache-to = [
    "type=local,dest=/tmp/.buildx-cache"
  ]

  output = ["type=docker"]
}

# Local build target (no registry push)
target "local" {
  inherits = ["app"]

  platforms = ["linux/amd64"]

  tags = [
    "${REGISTRY}:local",
    "${REGISTRY}:local-${formatdate("YYYY-MM-DD", timestamp())}"
  ]

  cache-from = [
    "type=local,src=/tmp/.buildx-cache"
  ]

  cache-to = [
    "type=local,dest=/tmp/.buildx-cache"
  ]

  output = ["type=docker"]
}
