# Runbook: Troubleshooting

## Gateway returning 503 on /ready

**Symptoms:** `/ready` returns 503, pods are running but not accepting traffic.

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -n ai-gateway

# Check logs for errors
kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway --tail=100

# Check Redis connectivity
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=ai-gateway-redis -n ai-gateway -o jsonpath='{.items[0].metadata.name}') -n ai-gateway -- redis-cli ping

# Check ConfigMap
kubectl get configmap ai-gateway-config -n ai-gateway -o yaml
```

**Resolution:**
- If Redis is down: restart Redis pod
- If config is wrong: fix ConfigMap and restart gateway pods
- If provider is misconfigured: check `providers` section in ConfigMap

---

## High Latency (p99 > threshold)

**Symptoms:** Grafana dashboard shows high p99 latency, slow responses.

**Diagnosis:**
```bash
# Check Grafana dashboard for provider latency breakdown
# Look at aigateway_provider_request_duration_seconds by provider

# Check circuit breaker state
curl http://localhost:8080/metrics | grep circuit_breaker_state

# Check rate limit metrics
curl http://localhost:8080/metrics | grep ratelimit

# Check upstream provider status pages
```

**Resolution:**
- If one provider is slow: check provider status page, consider enabling fallback
- If circuit breaker is open: provider may be degraded, wait for half-open or restart
- If rate limiter is throttling: adjust `requests_per_minute` in ConfigMap

---

## 429 Too Many Requests

**Symptoms:** Clients receiving 429 responses.

**Diagnosis:**
```bash
# Check rate limit rejection metrics
curl http://localhost:8080/metrics | grep ratelimit_rejected

# Check which key type is hitting limits (auth/key/ip)
```

**Resolution:**
- Adjust `requests_per_minute` and `burst` in ConfigMap
- If legitimate traffic: increase limits
- If abuse: investigate client behavior

---

## Circuit Breaker Stuck Open

**Symptoms:** Fallback always used, provider never retried.

**Diagnosis:**
```bash
# Check circuit breaker state
curl http://localhost:8080/metrics | grep circuit_breaker_state

# Check upstream provider health
kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway | grep "upstream error"
```

**Resolution:**
- Check upstream provider status page
- Wait for `open_timeout` to elapse (half-open probe)
- Force recovery: restart gateway pods
- Tune `failure_threshold` and `open_timeout` in ConfigMap

---

## OOMKilled Pods

**Symptoms:** Pods crash with OOMKilled status.

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -n ai-gateway | grep OOMKilled

# Check memory usage
kubectl top pod -n ai-gateway

# Check current limits
kubectl get deployment ai-gateway -n ai-gateway -o jsonpath='{.spec.template.spec.containers[0].resources}'
```

**Resolution:**
- Increase `memory` limits in deployment.yml
- Check for memory leaks in upstream response handling
- Reduce concurrent connections if needed

---

## Redis Connection Refused

**Symptoms:** Gateway logs show Redis connection errors.

**Diagnosis:**
```bash
# Check Redis pod status
kubectl get pods -l app.kubernetes.io/name=ai-gateway-redis -n ai-gateway

# Check Redis logs
kubectl logs -l app.kubernetes.io/name=ai-gateway-redis -n ai-gateway

# Check Redis service
kubectl get service ai-gateway-redis -n ai-gateway

# Test connectivity from gateway pod
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=ai-gateway -n ai-gateway -o jsonpath='{.items[0].metadata.name}') -n ai-gateway -- wget -qO- http://ai-gateway-redis:6379
```

**Resolution:**
- Restart Redis pod if it's down
- Check `redis_url` in ConfigMap matches service DNS
- Verify Redis PVC is bound (if using StatefulSet)

---

## Pod Not Starting

**Symptoms:** Pods in `CrashLoopBackOff` or `ImagePullBackOff`.

**Diagnosis:**
```bash
# Check pod events
kubectl describe pod <pod-name> -n ai-gateway

# Check logs
kubectl logs <pod-name> -n ai-gateway --previous
```

**Resolution:**
- ImagePullBackOff: check image name, registry auth, network
- CrashLoopBackOff: check logs for config errors, missing secrets
- Create namespace and secrets if not applied yet
