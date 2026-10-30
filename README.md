
**Reliability mechanisms:**
- **Idempotency:** `Idempotency-Key` (UUID) is unique in `votes`. Repeating an accepted vote with the same key and identical data returns the existing result (`200`, `replayed: true`); a new vote returns `201`. Reusing the key for different data or voting again with another key returns `409`.
- **Atomicity:** participation, vote insertion and counter increment run in one PostgreSQL transaction. Row locking coordinates voting with poll closing.
- **Timeouts:** HTTP server header/read/write/idle limits: 3/5/7/60 seconds; vote and poll creation handlers use 5-second contexts, results use 2 seconds.
- **Retries:** the Go client retries transient `429/502/503/504` responses and network errors up to **3 attempts**, with 100/200 ms backoff and the **same idempotency key**.


## Performance methodology

- Tool: **k6**, constant arrival rate, **60 seconds per scenario**, local Go server and PostgreSQL.
- Voting: **12** and **120 RPS** (different test users per vote); results reading: **25** and **250 RPS**.
- SIS #1 latency objectives: vote **p99 < 300 ms**, results **p99 < 200 ms**.
- Metrics: HTTP request duration (ms), actual RPS, error rate, dropped iterations, and serialized **JSON body** sizes (bytes).
- The saved JSON summaries are under `benchmarks/results/`. `benchmarks/vote.js`, `results.js`, and `protocol.js` contain the test scripts.

Example commands (replace `POLL_ID` and `OPTION_ID` with real IDs):

```powershell
k6 run -e BASE_URL=https://localhost:8443 -e POLL_ID=3 -e RPS=250 benchmarks/results.js
k6 run -e BASE_URL=https://localhost:8443 -e POLL_ID=3 -e OPTION_ID=8 -e RPS=120 -e USER_OFFSET=1000 benchmarks/vote.js
```

To force **HTTP/1.1 over HTTPS** in the Go server, launch it from PowerShell with `$env:GODEBUG="http2server=0"; go run ./cmd/server`. For **HTTP/2 over HTTPS**, remove that setting and restart: `Remove-Item Env:\GODEBUG -ErrorAction SilentlyContinue; go run ./cmd/server`. Check actual negotiation with:

```powershell
k6 run -e BASE_URL=https://localhost:8443 -e EXPECTED_PROTO=HTTP/2.0 benchmarks/protocol.js
```

Set `EXPECTED_PROTO=HTTP/1.1` for the HTTP/1.1 server.

## Measured results

**HTTPS runs**; figures below come from the archived k6 summaries (`h1-tls-*` and `h2-*`). Latency in **ms**. All eight scenarios reported **0% HTTP errors** and **0 dropped iterations**.

| Operation | Target RPS | HTTP/1.1 actual RPS | HTTP/1.1 avg | HTTP/1.1 p99 | HTTP/2 actual RPS | HTTP/2 avg | HTTP/2 p99 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Vote | 12 | 12.02 | 5.65 | 13.48 | 12.02 | 4.39 | 21.97 |
| Vote | 120 | 120.01 | 3.90 | 13.73 | 119.99 | 5.07 | 48.03 |
| Read results | 25 | 25.00 | 2.31 | 10.32 | 25.02 | 2.12 | 15.06 |
| Read results | 250 | 249.93 | 2.15 | 19.00 | 250.01 | 3.80 | 70.65 |

**JSON payload sizes (average, bytes):**

| Operation | HTTP/1.1 request | HTTP/1.1 response | HTTP/2 request | HTTP/2 response |
|---|---:|---:|---:|---:|
| Vote, 12 RPS | 28.86 | 80.00 | 30.00 | 79.00 |
| Vote, 120 RPS | 30.00 | 80.00 | 31.00 | 79.88 |
| Read results, 25/250 RPS | — | 344.00 | — | 342.00 |

These are **JSON body sizes, not full network-packet sizes**. Differences reflect varying IDs/counts in test data, not an HTTP/2 reduction in JSON body size. HTTP/2 header compression is not captured by these payload metrics.

**Interpretation:** both versions met the specified load and p99 objectives in these local runs. HTTP/1.1 showed lower p99 in all four scenarios; HTTP/2 showed slightly lower mean latency in the two normal-load scenarios. This is **not a general proof** that HTTP/1.1 is faster: only one run per scenario is recorded, test data differed slightly, and protocol negotiation evidence is not saved with the result summaries. Repeat matched tests 3–5 times before drawing a stronger conclusion. The original plain-HTTP baseline is excluded from the HTTPS comparison.

## When other protocols are preferable

| Technology | Best fit | Trade-off |
|---|---|---|
| **REST/JSON (HTTP/1.1 or HTTP/2)** | Public CRUD APIs, browser clients, simple integrations | Larger, text-based messages; conventional request/response pattern |
| **HTTP/2** | Many concurrent HTTP requests between the same endpoints | Multiplexing and header compression help some workloads, not automatically each small request |
| **gRPC + Protobuf** | Typed, high-frequency service-to-service RPCs, polyglot microservices, streaming | Schema/code generation and browser integration add complexity |
| **WebSocket** | Live collaborative apps, chat, bidirectional games and dashboards | Persistent connections and scaling/reconnection management |
| **SSE** | One-way live results updates from server to browser | Not bidirectional |

**Choice for this project:** REST/JSON is sufficient for poll creation, voting and result reads at the tested load. SSE is the simpler option for pushing live result updates; use WebSocket if real-time **two-way** interaction becomes necessary. gRPC is a reasonable future internal-service option but was **not implemented or measured** in SIS #2. HTTP/2 was evaluated as a transport for the existing REST endpoints.

