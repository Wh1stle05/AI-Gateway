# Runbook: Deployment

## Prerequisites

- `kubectl` access to the cluster
- Namespace created: `kubectl apply -f k8s/namespace.yml`
- Secrets configured: edit `k8s/secret.yml` then `kubectl apply -f k8s/secret.yml`

## Initial Deployment

```bash
# Apply all manifests
kubectl apply -f k8s/namespace.yml
kubectl apply -f k8s/secret.yml
kubectl apply -f k8s/configmap.yml
kubectl apply -f k8s/redis/
kubectl apply -f k8s/deployment.yml
kubectl apply -f k8s/service.yml
kubectl apply -f k8s/hpa.yml
kubectl apply -f k8s/pdb.yml

# Verify
kubectl get pods -n ai-gateway
kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway --tail=50
```

## Deploying a New Version

```bash
# 1. Tag the release
git tag v1.x.x
git push --tags

# 2. CI builds and pushes Docker image to GHCR

# 3. Update image tag
kubectl set image deployment/ai-gateway \
  gateway=ghcr.io/wh1stle05/ai-gateway:v1.x.x \
  -n ai-gateway

# 4. Watch rollout
kubectl rollout status deployment/ai-gateway -n ai-gateway
```

## Rollback

```bash
# Undo the last rollout
kubectl rollout undo deployment/ai-gateway -n ai-gateway

# Verify
kubectl rollout status deployment/ai-gateway -n ai-gateway
kubectl get pods -n ai-gateway
```

## Configuration Changes

```bash
# 1. Edit the ConfigMap
kubectl edit configmap ai-gateway-config -n ai-gateway

# 2. Restart pods to pick up new config
kubectl rollout restart deployment/ai-gateway -n ai-gateway

# 3. Verify
kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway --tail=20
```

## Secret Rotation

```bash
# Update the secret
kubectl create secret generic ai-gateway-secrets \
  --from-literal=OPENAI_API_KEY=sk-new-key \
  --from-literal=GATEWAY_API_KEY=new-gw-key \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to pick up new secrets
kubectl rollout restart deployment/ai-gateway -n ai-gateway
```

## Verification Checklist

- [ ] All pods are `Running` and `Ready`
- [ ] `/health` returns 200: `kubectl exec -it <pod> -- wget -qO- http://localhost:8080/health`
- [ ] `/ready` returns 200
- [ ] `/metrics` returns Prometheus metrics
- [ ] Logs show no errors
- [ ] HPA is active: `kubectl get hpa -n ai-gateway`
