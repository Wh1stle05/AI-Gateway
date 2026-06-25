# Runbook: Scaling

## Horizontal Scaling (HPA)

### Verify HPA Status

```bash
# Check HPA
kubectl get hpa -n ai-gateway

# Detailed HPA info
kubectl describe hpa ai-gateway -n ai-gateway

# Check current vs desired replicas
kubectl get deployment ai-gateway -n ai-gateway
```

### Tune HPA

If HPA is oscillating (scaling up and down rapidly):

```bash
# Increase stabilization windows in hpa.yml
# scaleUp: stabilizationWindowSeconds: 120
# scaleDown: stabilizationWindowSeconds: 600

# Or adjust metrics thresholds
# targetCPUUtilizationPercentage: 80 (increase to reduce sensitivity)
```

### Scale Based on Prometheus Metrics

With `prometheus-adapter` installed:

```yaml
# Add to hpa.yml metrics section:
- type: Pods
  pods:
    metric:
      name: aigateway_http_requests_in_flight
    target:
      type: AverageValue
      averageValue: "50"
```

## Vertical Scaling

```bash
# Edit deployment resource limits
kubectl edit deployment ai-gateway -n ai-gateway

# Or apply updated manifest
kubectl apply -f k8s/deployment.yml

# Pods will restart with new limits
kubectl rollout status deployment/ai-gateway -n ai-gateway
```

## Redis Scaling

Redis is single-threaded. For high throughput:

1. **Vertical first:** Increase Redis CPU/memory limits
2. **Monitor memory:** `redis-cli info memory`
3. **Horizontal:** Consider Redis Cluster for >10k RPM

```bash
# Check Redis memory usage
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=ai-gateway-redis -n ai-gateway -o jsonpath='{.items[0].metadata.name}') -n ai-gateway -- redis-cli info memory | grep used_memory_human
```

## Load Testing Before Scaling

```bash
# Run k6 load test
k6 run scripts/loadtest.js --vus 100 --duration 5m

# Or via Docker
docker run --rm -i grafana/k6 run - < scripts/loadtest.js

# Monitor Grafana during test
# Watch HPA: kubectl get hpa -n ai-gateway -w
```

## Cost Optimization

- Review HPA `minReplicas` — don't over-provision
- Check if mock provider is used in production (wasted resources)
- Review Prometheus retention (`--storage.tsdb.retention.time`)
- Use spot/preemptible nodes for non-critical workloads
- Right-size resource requests based on actual usage
