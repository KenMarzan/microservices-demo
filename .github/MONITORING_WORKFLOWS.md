# Monitoring GitHub Actions Setup Guide

This guide explains the newly added monitoring workflows for the microservices-demo project.

## 🚀 Quick Start

The three workflows are now active:

1. **ci.yml** - Runs on every push/PR (full test suite)
2. **monitoring-test.yml** - Runs when monitoring files change (metrics validation)
3. **docker-build.yml** - Runs on main branch (builds and pushes to registry)

All workflows are **already configured and ready to use**. No setup needed!

---

## 📋 What Each Workflow Does

### CI Workflow (`ci.yml`)
- Builds all Docker images
- Starts full microservices stack + monitoring
- Verifies service health
- Validates Prometheus can scrape metrics
- Runs 30-second load test
- Confirms metrics were collected
- Takes ~5-7 minutes

**Runs on:**
- Every push to `main` or `develop`
- Every pull request to `main` or `develop`

**Status:** ✅ Ready to run

---

### Monitoring Test Workflow (`monitoring-test.yml`)
- Validates Prometheus health
- Checks all scrape targets
- Verifies metric collection
- Tests Grafana connectivity
- Measures query performance
- Takes ~3-4 minutes

**Runs on:**
- Push to `main/develop` with changes to:
  - `monitoring/` directory
  - `docker-compose.monitoring.yml`
  - This workflow file
- Pull requests with monitoring changes

**Status:** ✅ Ready to run

---

### Docker Build Workflow (`docker-build.yml`)
- Builds 10 microservices
- Pushes to GitHub Container Registry (GHCR)
- Tags with commit SHA and versions
- Takes ~10-15 minutes

**Runs on:**
- Push to `main` branch
- Git tags matching `v*` (releases)
- Manual trigger (workflow_dispatch)

**Status:** ✅ Ready to run

---

## 🔍 Viewing Workflow Runs

### In GitHub UI:
1. Go to your repository
2. Click **Actions** tab at the top
3. Select workflow from left sidebar
4. Click on a run to see details

### View specific steps:
- Click a workflow run
- Expand the step you want to see
- View logs and timing information

### Check workflow status:
```bash
# View all workflow runs (requires GitHub CLI)
gh run list

# View specific workflow
gh run list -w ci.yml

# View a specific run
gh run view <run-id>
```

---

## 🛠️ Local Testing (Replicate CI)

To test before pushing:

```bash
# 1. Build Docker images
docker-compose build --no-cache

# 2. Start services
docker-compose up -d
sleep 20

# 3. Start monitoring
docker-compose -f docker-compose.monitoring.yml up -d
sleep 10

# 4. Run health checks (same as CI does)
curl -f http://localhost:8080              # Frontend
curl -f http://localhost:5050              # Checkout
curl http://localhost:9090/-/healthy       # Prometheus

# 5. Check Prometheus targets
curl -s http://localhost:9090/api/v1/targets | \
  jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# 6. Verify metrics
curl -s http://localhost:9090/api/v1/query?query=up | \
  jq '.data.result[] | {job: .metric.job, value: .value[1]}'

# 7. Cleanup
docker-compose -f docker-compose.monitoring.yml down -v
docker-compose down -v
```

---

## ❌ Troubleshooting Workflow Failures

### Port Already in Use
**Error:** `Address already in use`

**Fix:**
```bash
docker-compose down -v
docker-compose -f docker-compose.monitoring.yml down -v
```

### Prometheus Targets Won't Come Up
**Error:** `health: "down"`

**Debug:**
```bash
# Check Prometheus logs
docker logs prometheus | tail -50

# Verify service is reachable from Prometheus container
docker exec prometheus curl -v http://frontend:8080/metrics

# Check Prometheus config
curl -s http://localhost:9090/api/v1/status/config | jq '.data.yaml' -r
```

### Metrics Not Collecting
**Error:** `up` metric returns empty array

**Debug:**
```bash
# Wait longer for scrapes to complete
sleep 30

# Query again
curl -s http://localhost:9090/api/v1/query?query=up | jq '.data.result'

# Check scrape history
curl -s http://localhost:9090/api/v1/targets | \
  jq '.data.activeTargets[] | {job: .labels.job, lastScrape: .lastScrape, lastError: .lastError}'
```

### Workflow Doesn't Trigger
**Error:** Workflow not running on push

**Debug:**
1. Check branch name (must be `main` or `develop`)
2. Ensure workflow file is in `.github/workflows/`
3. Verify YAML syntax: `yamllint .github/workflows/`
4. Check Actions permissions: Settings → Actions → General

---

## 📊 Monitoring Workflow Performance

Expected runtime:
- **First run:** ~15 minutes (includes Docker layer builds)
- **Subsequent runs:** ~5 minutes (uses cached layers)
- **docker-build.yml:** ~10-15 minutes (builds 10 services)

GitHub Actions includes:
- 20 GB of free monthly actions minutes (Ubuntu)
- After that: $0.24 per minute

---

## 🔐 Security & Secrets

### What Workflows Can Access:
- `GITHUB_TOKEN` - Automatically provided, can't be exposed
- Repository code - Workflows have read access
- Artifacts - Can upload/download within 90 days

### What Workflows CANNOT Access:
- Your personal GitHub token
- Private repository credentials
- Third-party API keys (unless added as secrets)

### To Add Secrets:
1. Go to Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add secret name and value
4. Use in workflow: `${{ secrets.SECRET_NAME }}`

---

## 📝 Customizing Workflows

### Modify CI Timing
Edit `.github/workflows/ci.yml`:
```yaml
# Increase wait time for services
- name: Start microservices stack
  run: |
    docker-compose up -d
    sleep 20  # <- Change this to 30 or 40 if services need more time
```

### Add New Health Check
Edit `.github/workflows/ci.yml`:
```yaml
- name: Check service health
  run: |
    curl -f http://localhost:8080          # existing
    curl -f http://localhost:5050          # existing
    curl -f http://localhost:3000          # <- Add Grafana check
```

### Modify Monitoring Tests
Edit `.github/workflows/monitoring-test.yml`:
```yaml
# Change query performance threshold
if [ "$SIMPLE" -gt 10000 ]; then  # <- Increase from 5000 to 10000
```

---

## 🚨 Common Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| Workflows don't run | Not on main/develop | Push to correct branch |
| "Port already in use" | Service still running | `docker-compose down -v` |
| Prometheus targets down | Network isolation | Check docker-compose networks |
| Metrics not collecting | Services don't export metrics | Add HTTP metrics endpoints |
| Grafana not reachable | Port conflict | Check port 3000 is free |
| Build takes 30min | First run, no cache | Subsequent runs are faster |
| Intermittent failures | Timing issues | Increase sleep times |

---

## ✅ Best Practices

✅ **DO:**
- Test locally before pushing
- Check workflow logs for errors
- Use semantic versioning for releases
- Run monitoring-test on every monitoring change
- Keep docker-compose files in sync

❌ **DON'T:**
- Push to main without workflow passing
- Ignore failed runs
- Manually edit images in registry
- Commit without local testing
- Modify prometheus.yml without testing

---

## 🔗 Useful Links

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [Docker in Actions](https://docs.github.com/en/actions/using-docker-with-workflows)
- [View Workflow Logs](https://docs.github.com/en/actions/monitoring-and-troubleshooting-workflows/viewing-workflow-run-history)

---

## Next Steps

1. **Test locally** - Run ci.yml commands manually
2. **Make a commit** - Push to trigger workflows
3. **View Actions tab** - Monitor workflow runs
4. **Add branch protection** - Require workflows pass before merge
5. **Set up Slack notifications** - Get alerts on failures

---

## Questions?

Check the main `.github/workflows/README.md` for infrastructure details, or refer to individual workflow file comments.
