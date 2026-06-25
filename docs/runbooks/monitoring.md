# Runbook: Monitoring & Alerting

## Prometheus Alerts

Alerting rules are deployed via `k8s/monitoring/prometheus-configmap.yml`.

### Alert: HighErrorRate

- **Expr:** `sum(rate(aigateway_chat_errors_total[5m])) / sum(rate(aigateway_chat_requests_total[5m])) > 0.05`
- **Duration:** 5 minutes
- **Severity:** Warning
- **Action:** Check upstream provider health, review error types in Grafana

### Alert: HighLatency

- **Expr:** `histogram_quantile(0.99, sum(rate(aigateway_http_request_duration_seconds_bucket[5m])) by (le)) > 2.0`
- **Duration:** 5 minutes
- **Severity:** Warning
- **Action:** Check provider latency breakdown, consider enabling fallback

### Alert: CircuitBreakerOpen

- **Expr:** `aigateway_circuit_breaker_state == 1`
- **Duration:** 2 minutes
- **Severity:** Critical
- **Action:** Check upstream provider status, wait for half-open or restart

### Alert: HighRateLimitRejection

- **Expr:** `sum(rate(aigateway_ratelimit_rejected_total[5m])) > 10`
- **Duration:** 5 minutes
- **Severity:** Warning
- **Action:** Adjust rate limits or investigate abusive clients

### Alert: GatewayDown

- **Expr:** `up{job="ai-gateway"} == 0`
- **Duration:** 1 minute
- **Severity:** Critical
- **Action:** Check pod status, restart if needed

## Grafana Dashboard

Access Grafana at `http://grafana:3000` (or via Ingress).

### Key Panels

| Panel | What to Look For |
|-------|------------------|
| Request Rate | Sudden drops or spikes |
| Latency (p99) | Values > 2s indicate issues |
| Error Rate | Any non-zero error types |
| Circuit Breaker | State changes to "Open" |
| Rate Limit | High rejection counts |
| Fallback | Frequent fallback activations |

### Creating Alerts from Dashboard

1. Open a panel
2. Click "Edit" → "Alert" tab
3. Set condition and notification channel
4. Save

## On-Call Escalation

1. **Check Grafana dashboard** for overall health
2. **Check Prometheus alerts** at `http://prometheus:9090/alerts`
3. **Check pod logs:** `kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway --tail=100`
4. **Follow troubleshooting runbook** for specific symptoms
5. **Escalate** if unable to resolve within 15 minutes

## Useful Commands

```bash
# Quick health check
kubectl get pods -n ai-gateway
curl http://localhost:8080/health
curl http://localhost:8080/ready

# View metrics
curl http://localhost:8080/metrics

# Tail logs
kubectl logs -l app.kubernetes.io/name=ai-gateway -n ai-gateway -f

# Check HPA
kubectl get hpa -n ai-gateway

# Check alerts
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=prometheus -n ai-gateway -o jsonpath='{.items[0].metadata.name}') -n ai-gateway -- wget -qO- http://localhost:9090/api/v1/alerts
```
