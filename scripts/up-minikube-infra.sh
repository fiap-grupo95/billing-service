#!/usr/bin/env bash
set -euo pipefail

PROFILE="eks-local"
APPLY_NEWRELIC="false"
RUN_SEED_JOB="false"

usage() {
  cat <<'EOF'
Uso: ./scripts/up-minikube-infra.sh [opções]

Opções:
  --profile <nome>       Profile do Minikube (padrão: eks-local)
  --apply-newrelic       Aplica os manifests de New Relic
  --run-seed-job         Executa o migration/seed job do os-service-api
  -h, --help             Exibe ajuda
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="${2:-}"
      if [[ -z "$PROFILE" ]]; then
        echo "Erro: informe um valor para --profile"
        exit 1
      fi
      shift 2
      ;;
    --apply-newrelic)
      APPLY_NEWRELIC="true"
      shift
      ;;
    --run-seed-job)
      RUN_SEED_JOB="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Opção inválida: $1"
      usage
      exit 1
      ;;
  esac
done

assert_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Erro: comando '$cmd' não encontrado no PATH."
    exit 1
  fi
}

apply_if_exists() {
  local file_path="$1"
  if [[ -f "$file_path" ]]; then
    kubectl apply -f "$file_path"
  fi
}

patch_image_pull_policy() {
  local namespace="$1"
  local deployment="$2"
  local container="$3"

  kubectl -n "$namespace" patch deployment "$deployment" --type strategic -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"$container\",\"imagePullPolicy\":\"IfNotPresent\"}]}}}}"
}

cleanup() {
  echo "==> Restaurando ambiente Docker local..."
  eval "$(minikube -p "$PROFILE" docker-env --shell bash -u)" || true
}
trap cleanup EXIT

assert_command minikube
assert_command kubectl
assert_command docker

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BILLING_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE_ROOT="$(cd "$BILLING_ROOT/.." && pwd)"
OS_ROOT="$WORKSPACE_ROOT/os-service-api"

if [[ ! -d "$OS_ROOT" ]]; then
  echo "Erro: pasta do os-service-api não encontrada em: $OS_ROOT"
  exit 1
fi

BILLING_DEPLOY_DIR="$BILLING_ROOT/infrastructure/k8s/deploy"
OS_DEPLOY_DIR="$OS_ROOT/internal/infrastructure/k8s/deploy"
OS_JOB_DIR="$OS_ROOT/internal/infrastructure/k8s/job-process"

if [[ ! -d "$BILLING_DEPLOY_DIR" ]]; then
  echo "Erro: pasta não encontrada: $BILLING_DEPLOY_DIR"
  exit 1
fi
if [[ ! -d "$OS_DEPLOY_DIR" ]]; then
  echo "Erro: pasta não encontrada: $OS_DEPLOY_DIR"
  exit 1
fi

echo "==> Iniciando Minikube (profile: $PROFILE)..."
minikube start -p "$PROFILE" --driver=docker

echo "==> Configurando Docker para usar daemon do Minikube..."
eval "$(minikube -p "$PROFILE" docker-env --shell bash)"

echo "==> Build imagem billing-service..."
pushd "$BILLING_ROOT" >/dev/null
docker build . --file Dockerfile --tag mandaapag03/billing-service:latest
popd >/dev/null

echo "==> Build imagem os-service-api..."
pushd "$OS_ROOT" >/dev/null
docker build . --file Dockerfile --tag mandaapag03/os-service-api:latest
popd >/dev/null

echo "==> Aplicando manifestos do billing-service..."
pushd "$BILLING_DEPLOY_DIR" >/dev/null
kubectl apply -f namespace.yml
kubectl apply -f secret-local.yml
kubectl apply -f configmap-local.yml
kubectl apply -f service-api.yml
kubectl apply -f deployment-api.yml
kubectl apply -f hpa-api.yml

if [[ "$APPLY_NEWRELIC" == "true" ]]; then
  kubectl apply -f newrelic-namespace.yml
  kubectl apply -f newrelic-secret.yml
  kubectl apply -f newrelic-configmap.yml
  kubectl apply -f newrelic-fluent-bit-configmap.yml
  kubectl apply -f newrelic-events-configmap.yml
  kubectl apply -f newrelic-rbac.yml
  kubectl apply -f newrelic-daemonset-infra.yml
  kubectl apply -f newrelic-daemonset-fluent-bit.yml
  kubectl apply -f newrelic-kube-state-metrics.yml
  kubectl apply -f newrelic-deployment-events.yml
fi
popd >/dev/null

echo "==> Aplicando manifestos do os-service-api..."
pushd "$OS_DEPLOY_DIR" >/dev/null
kubectl apply -f namespace.yml
kubectl apply -f secret-local.yml
kubectl apply -f configmap-local.yml
apply_if_exists "pvc.yml"
apply_if_exists "service-db.yml"
apply_if_exists "deployment-db.yml"
kubectl apply -f service-api.yml
kubectl apply -f deployment-api.yml
kubectl apply -f hpa-api.yml
popd >/dev/null

if [[ "$RUN_SEED_JOB" == "true" ]]; then
  echo "==> Executando seed job do os-service-api..."
  pushd "$OS_JOB_DIR" >/dev/null
  kubectl -n os-service-api delete job seed-job --ignore-not-found=true
  kubectl apply -f migration-job.yml
  popd >/dev/null
fi

echo "==> Ajustando deploys para usar imagens locais (IfNotPresent)..."
kubectl -n billing-service set image deployment/billing-service-deployment billing-service=mandaapag03/billing-service:latest
patch_image_pull_policy "billing-service" "billing-service-deployment" "billing-service"

kubectl -n os-service-api set image deployment/os-service-api-deployment os-service-api=mandaapag03/os-service-api:latest
patch_image_pull_policy "os-service-api" "os-service-api-deployment" "os-service-api"

echo "==> Aguardando rollouts..."
kubectl -n billing-service rollout status deployment/billing-service-deployment --timeout=240s
kubectl -n os-service-api rollout status deployment/os-service-api-deployment --timeout=240s

if kubectl -n os-service-api get deployment mongodb -o name >/dev/null 2>&1; then
  kubectl -n os-service-api rollout status deployment/mongodb --timeout=240s
fi

echo "==> Resumo de recursos"
kubectl get ns billing-service os-service-api
kubectl -n billing-service get all
kubectl -n os-service-api get all

echo
echo "Concluído. Infra aplicada no Minikube profile '$PROFILE'."
