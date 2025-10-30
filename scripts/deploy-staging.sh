#!/usr/bin/env bash
set -euo pipefail

# Guard: skip entirely if kubeconfig not provided
if [[ -z "${KUBECONFIG_CONTENTS:-}" ]]; then
  echo "KUBECONFIG_CONTENTS not set; skipping deploy"
  exit 0
fi

# Prepare kubeconfig
mkdir -p "${HOME}/.kube"
echo "${KUBECONFIG_CONTENTS}" > "${HOME}/.kube/config"
chmod 600 "${HOME}/.kube/config"

# Tools install (local user bin)
BIN_DIR="${HOME}/bin"
mkdir -p "${BIN_DIR}"
export PATH="${BIN_DIR}:${PATH}"

# Install kubectl v1.30.0 if missing
if ! command -v kubectl >/dev/null 2>&1; then
  echo "Installing kubectl v1.30.0"
  curl -fsSL -o "${BIN_DIR}/kubectl" "https://storage.googleapis.com/kubernetes-release/release/v1.30.0/bin/linux/amd64/kubectl"
  chmod +x "${BIN_DIR}/kubectl"
fi

# Install helm v3.14.4 if missing
if ! command -v helm >/dev/null 2>&1; then
  echo "Installing helm v3.14.4"
  curl -fsSL -o /tmp/helm.tgz "https://get.helm.sh/helm-v3.14.4-linux-amd64.tar.gz"
  tar -xzf /tmp/helm.tgz -C /tmp
  mv /tmp/linux-amd64/helm "${BIN_DIR}/helm"
  chmod +x "${BIN_DIR}/helm"
fi

# Inputs
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY_URL="${REGISTRY_URL:-}"

if [[ -z "${REGISTRY_URL}" ]]; then
  echo "REGISTRY_URL not set; cannot form image repo"
  exit 1
fi

# Deploy via Helm
helm upgrade --install aura ./deploy/helm/aura \
  --namespace aura --create-namespace \
  --set image.backend.repository="${REGISTRY_URL}/aura-backend" \
  --set image.backend.tag="${IMAGE_TAG}" \
  --set image.frontend.repository="${REGISTRY_URL}/aura-frontend" \
  --set image.frontend.tag="${IMAGE_TAG}"
