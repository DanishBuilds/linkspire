<!-- test/load/README.md -->
# Load tests (k6)

Scripts that encode the platform SLOs as machine-checked thresholds.

| Script | Target | Thresholds |
|--------|--------|-----------|
| `smoke.js` | liveness of redirect + API | all checks pass |
| `redirect_hotpath.js` | redirect hot path | p99 < 30 ms, p95 < 15 ms, errors < 0.1% |
| `api_load.js` | management API | p95 < 200 ms, errors < 1% |

## Run

```bash
# Against the local compose stack (deploy/docker/compose.yaml):
k6 run -e REDIRECT_URL=http://localhost:9090 -e API_URL=http://localhost:9091 test/load/smoke.js
k6 run -e BASE_URL=http://localhost:8080 -e CODES=<comma,separated,codes> test/load/redirect_hotpath.js
k6 run -e API_URL=http://localhost:8081 test/load/api_load.js
```

The 30 ms p99 is the origin SLO (`docs/architecture/01`, `02`). A localhost run will not
reproduce production geography; run `redirect_hotpath.js` from load generators in the target
region against a real deployment to validate the SLO, and archive the k6 summary JSON
(`k6 run --summary-export=results/<date>.json ...`) under `results/`.
