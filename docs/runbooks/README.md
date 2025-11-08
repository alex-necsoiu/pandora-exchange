# SRE Runbooks

> **Operational procedures for Pandora Exchange User Service**  
> **Last Updated:** November 8, 2025

---

## Overview

This directory contains runbooks for Site Reliability Engineers (SREs) and operators managing the Pandora Exchange User Service in production.

**Purpose:**
- Provide step-by-step procedures for common operations
- Enable quick incident response
- Ensure consistent deployment practices
- Document troubleshooting workflows

**Audience:**
- SREs
- DevOps Engineers
- On-call Engineers
- Operations Team

---

## Runbook Index

| Runbook | Purpose | When to Use |
|---------|---------|-------------|
| **[Deployment](deployment.md)** | Deploy User Service to Kubernetes | Releases, hotfixes, rollbacks |
| **[Debugging](debugging.md)** | Troubleshoot production issues | Incidents, performance issues, errors |
| **[Incident Response](incident-response.md)** | Handle security incidents | Security alerts, suspicious activity |

---

## Quick Reference

### Emergency Contacts

| Role | Contact | Hours |
|------|---------|-------|
| **On-Call SRE** | Slack: @oncall-sre | 24/7 |
| **Security Team** | Slack: #pandora-security | 24/7 |
| **Engineering Lead** | [Name] | Business hours |
| **Database Admin** | Slack: @dba-oncall | 24/7 |

### Critical Resources

- **Grafana Dashboards:** https://grafana.pandora.exchange/d/user-service
- **Logs (Loki):** https://grafana.pandora.exchange/explore?datasource=loki
- **Traces (Jaeger):** https://jaeger.pandora.exchange
- **Kubernetes Dashboard:** https://k8s-dashboard.pandora.exchange
- **Vault UI:** https://vault.pandora.exchange
- **Status Page:** https://status.pandora.exchange

### Quick Commands

```bash
# Get service status
kubectl get pods -n pandora-exchange -l app=user-service

# View logs (last 100 lines)
kubectl logs -n pandora-exchange -l app=user-service --tail=100

# Describe pod (events, status)
kubectl describe pod -n pandora-exchange <pod-name>

# Execute command in pod
kubectl exec -it -n pandora-exchange <pod-name> -- /bin/sh

# Port forward for local debugging
kubectl port-forward -n pandora-exchange svc/user-service 8080:8080

# Scale deployment
kubectl scale deployment -n pandora-exchange user-service --replicas=5

# Restart deployment (rolling restart)
kubectl rollout restart deployment -n pandora-exchange user-service

# Check rollout status
kubectl rollout status deployment -n pandora-exchange user-service
```

---

## Runbook Conventions

### Severity Levels

**P0 - Critical:**
- Service completely down
- Data loss or corruption
- Security breach
- Response time: Immediate (< 5 minutes)

**P1 - High:**
- Partial service degradation
- Elevated error rates (> 5%)
- Failed deployments
- Response time: < 15 minutes

**P2 - Medium:**
- Non-critical feature impaired
- Performance degradation
- Configuration issues
- Response time: < 1 hour

**P3 - Low:**
- Minor issues
- Cosmetic problems
- Documentation updates
- Response time: Best effort

### Status Codes

- 🟢 **Healthy:** All systems operational
- 🟡 **Degraded:** Partial functionality
- 🔴 **Down:** Service unavailable
- ⚪ **Maintenance:** Planned downtime

### Command Annotations

```bash
# ✅ Safe in production
kubectl get pods

# ⚠️  Use with caution (reads sensitive data)
kubectl get secret -n pandora-exchange user-service-secrets -o yaml

# ❌ DANGEROUS - Never run in production
kubectl delete namespace pandora-exchange
```

---

## Common Operations

### Health Checks

**Check service health:**
```bash
# HTTP health check
curl https://api.pandora.exchange/health

# Expected response:
# {"status":"ok","version":"1.2.3","timestamp":"2025-11-08T10:30:00Z"}

# Check from inside cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://user-service.pandora-exchange.svc.cluster.local:8080/health
```

**Check database connectivity:**
```bash
# Port forward to database
kubectl port-forward -n pandora-exchange svc/postgres 5432:5432

# Test connection (from local machine)
psql -h localhost -U pandora_user -d pandora_db -c "SELECT version();"
```

**Check Vault connectivity:**
```bash
# Check Vault status
kubectl exec -it -n vault vault-0 -- vault status

# Check policy
kubectl exec -it -n vault vault-0 -- vault policy read pandora-user-service
```

### Log Access

**View application logs:**
```bash
# Tail logs (follow mode)
kubectl logs -n pandora-exchange -l app=user-service -f

# Search for errors
kubectl logs -n pandora-exchange -l app=user-service | grep -i error

# Logs for specific pod
kubectl logs -n pandora-exchange user-service-7d8f9c6b5-abcde

# Previous container logs (after crash)
kubectl logs -n pandora-exchange user-service-7d8f9c6b5-abcde --previous
```

**Loki queries (Grafana):**
```logql
# All user-service logs
{app="user-service", namespace="pandora-exchange"}

# Errors only
{app="user-service"} |= "level=error"

# Specific user
{app="user-service"} |= "user_id=550e8400-e29b-41d4-a716-446655440000"

# Trace ID
{app="user-service"} |= "trace_id=4bf92f3577b34da6a3ce929d0e0e4736"

# Performance (slow requests > 1s)
{app="user-service"} | json | duration > 1000
```

### Metrics & Monitoring

**Key Metrics:**

| Metric | Threshold | Alert |
|--------|-----------|-------|
| **Error Rate** | > 1% | P1 |
| **Latency (p95)** | > 500ms | P2 |
| **CPU Usage** | > 80% | P2 |
| **Memory Usage** | > 85% | P1 |
| **Pod Restarts** | > 5/hour | P1 |
| **Database Connections** | > 90% pool | P1 |

**Prometheus queries:**
```promql
# Request rate (requests/sec)
rate(http_requests_total{app="user-service"}[5m])

# Error rate (percentage)
sum(rate(http_requests_total{app="user-service",status=~"5.."}[5m])) 
  / sum(rate(http_requests_total{app="user-service"}[5m])) * 100

# Latency (p95)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# CPU usage
rate(container_cpu_usage_seconds_total{pod=~"user-service.*"}[5m])

# Memory usage
container_memory_working_set_bytes{pod=~"user-service.*"}
```

---

## Escalation Paths

### Incident Response Flow

```
1. Detect Issue
   ↓
2. Assess Severity (P0/P1/P2/P3)
   ↓
3. Notify Team (Slack, PagerDuty)
   ↓
4. Follow Runbook (this directory)
   ↓
5. Mitigate (rollback, scale, etc.)
   ↓
6. Monitor Recovery
   ↓
7. Post-Mortem (P0/P1 only)
```

### Escalation Matrix

| Issue Type | First Responder | Escalate To | Timeline |
|------------|----------------|-------------|----------|
| **Service Down** | On-call SRE | Engineering Lead | 15 min |
| **Database Issue** | On-call SRE | DBA Team | 10 min |
| **Security Incident** | On-call SRE | Security Team | Immediate |
| **Deployment Failure** | Deployer | SRE Team | 5 min |
| **Performance Degradation** | On-call SRE | Engineering Team | 30 min |

---

## Maintenance Windows

### Scheduled Maintenance

**Weekly:**
- Sunday 2:00 AM - 4:00 AM UTC (low traffic window)
- Use for non-critical updates, configuration changes

**Monthly:**
- First Sunday 0:00 AM - 6:00 AM UTC
- Use for database maintenance, major updates

**Procedure:**
1. Announce maintenance 72 hours in advance (email, status page)
2. Update status page to "Maintenance"
3. Enable maintenance mode (optional)
4. Perform maintenance
5. Verify service health
6. Update status page to "Operational"
7. Send completion notification

### Emergency Maintenance

**Criteria:**
- Critical security patch
- Data corruption risk
- Service completely down

**Procedure:**
1. Notify team immediately (Slack, PagerDuty)
2. Update status page
3. Perform fix (follow relevant runbook)
4. Verify health
5. Update status page
6. Post-mortem within 24 hours

---

## Disaster Recovery

### Recovery Time Objective (RTO)

**Target:** 1 hour (from incident to full recovery)

**Recovery Point Objective (RPO):**
**Target:** 15 minutes (maximum data loss)

### Backup Strategy

**Database:**
- Automated backups every 6 hours
- Point-in-time recovery (PITR) enabled
- Retention: 30 days
- Off-site replication to secondary region

**Configuration:**
- GitOps (all manifests in git)
- Secrets in Vault (backed up)

**Recovery Procedure:**
See [Incident Response](incident-response.md) → "Database Failure" section

---

## Post-Mortem Template

**Incident:** [Brief description]  
**Date:** YYYY-MM-DD  
**Duration:** [Start time - End time]  
**Severity:** P0/P1/P2/P3  
**Impact:** [Users affected, revenue impact, etc.]

**Timeline:**
- HH:MM - Event occurred
- HH:MM - Alert triggered
- HH:MM - Investigation started
- HH:MM - Root cause identified
- HH:MM - Mitigation applied
- HH:MM - Service restored

**Root Cause:**
[Detailed explanation]

**Resolution:**
[What fixed it]

**Action Items:**
- [ ] Improve monitoring (owner, due date)
- [ ] Update runbook (owner, due date)
- [ ] Code fix (ticket link)
- [ ] Infrastructure change (ticket link)

**Lessons Learned:**
- What went well
- What could be improved
- Prevention measures

---

## Runbook Maintenance

**Review Schedule:** Quarterly

**Owners:**
- **Deployment:** DevOps Team
- **Debugging:** SRE Team
- **Incident Response:** Security Team + SRE Team

**Update Process:**
1. Make changes via Pull Request
2. Test procedures in sandbox environment
3. Get approval from runbook owner
4. Merge and announce changes in #pandora-ops

---

## References

- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System architecture
- [User Service Documentation](../services/user-service.md) - Service details
- [Error Handling](../errors.md) - Error codes and meanings
- [Security Overview](../security/README.md) - Security procedures
- [Vault Integration](../../VAULT_INTEGRATION.md) - Vault setup

---

**Last Updated:** November 8, 2025  
**Maintained By:** SRE Team  
**Next Review:** February 2026
