# New Relic - Deploy via Manifestos Kubernetes

Este diretório contém os manifestos Kubernetes para deploy do New Relic Infrastructure e coleta de logs.

## Arquivos

- `newrelic-namespace.yml` - Namespace para recursos do New Relic
- `newrelic-secret.yml` - Secret com a License Key
- `newrelic-configmap.yml` - Configurações gerais do cluster
- `newrelic-rbac.yml` - Service Account, ClusterRole e ClusterRoleBinding
- `newrelic-daemonset-infra.yml` - Infrastructure Agent (métricas dos nodes)
- `newrelic-fluent-bit-configmap.yml` - Configuração do Fluent Bit
- `newrelic-daemonset-fluent-bit.yml` - Fluent Bit para coleta de logs
- `newrelic-kube-state-metrics.yml` - Kube State Metrics
- `newrelic-events-configmap.yml` - Configuração do Events Collector
- `newrelic-deployment-events.yml` - Events Collector

## Pré-requisitos

1. Cluster Kubernetes rodando
2. `kubectl` configurado e conectado ao cluster
3. License Key do New Relic ([obtenha aqui](https://one.newrelic.com/admin-portal/api-keys/home))

## Instalação

### Passo 1: Configurar License Key

Edite o arquivo `newrelic-secret.yml` e substitua `YOUR_LICENSE_KEY_HERE` pela sua License Key:

```yaml
stringData:
  license: "sua_license_key_aqui"
```

### Passo 2: Aplicar os Manifestos

#### Aplicação Manual

```bash
# 1. Namespace
kubectl apply -f newrelic-namespace.yml

# 2. Secret
kubectl apply -f newrelic-secret.yml

# 3. ConfigMaps
kubectl apply -f newrelic-configmap.yml
kubectl apply -f newrelic-fluent-bit-configmap.yml
kubectl apply -f newrelic-events-configmap.yml

# 4. RBAC
kubectl apply -f newrelic-rbac.yml

# 5. Infrastructure Agent
kubectl apply -f newrelic-daemonset-infra.yml

# 6. Fluent Bit (Logs)
kubectl apply -f newrelic-daemonset-fluent-bit.yml

# 7. Kube State Metrics
kubectl apply -f newrelic-kube-state-metrics.yml

# 8. Events Collector
kubectl apply -f newrelic-deployment-events.yml
```
