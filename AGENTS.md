# Robot

Service that executes swapping and batching on the Testnet/HLF platform.

## Build & Dev

```shell
go build -ldflags="-X 'main.AppInfoVer=${VERSION}'"   # compile
go fix ./...                                           # fix deprecated API usage
go test ./... -short                                   # unit tests only
```

- **No Makefile, no task runner.** All commands are plain Go toolchain.
- **No `go.work`.** Two separate modules: root (`github.com/anoideaopen/robot`) and `test/integration/`.
- **HLF SDK fork:** `github.com/hyperledger/fabric-sdk-go` is replaced by `github.com/anoideaopen/fabric-sdk-go v0.1.1` in both `go.mod` files.
- **Config:** YAML loaded via `spf13/viper`. Priority: `-c <path>` flag > `ROBOT_CONFIG` env > `./config.yaml` > `/etc/config.yaml`. All keys overridable via `ROBOT_`-prefixed env vars.

## CI Pipeline (`.github/workflows/go.yml`)

Runs on every push/PR. Order matters:

1. check-cyrillic-comments — no Cyrillic in source files
2. validate-go — `go mod tidy && go fmt ./...` (fails on dirty diff)
3. golangci-lint — `golangci/golangci-lint-action@v9`, config in `.golangci.yml` (59+ linters, v2 format, Go 1.27)
4. unit test — `go test -count 1 ./...`
5. coverage — `vladopajic/go-test-coverage@v2.8.1` with `.testcoverage.yml` (thresholds all 0, excludes `*.pb.go`)
6. integration test — ginkgo matrix across 6 suites (`swap`, `storage`, `chcol1`, `chcol2`, `chcol3`, `chexec`)

## Testing

### Unit tests

- Framework: `testing` + `github.com/stretchr/testify/require`
- Stubs: hand-written via `github.com/anoideaopen/common-component/testshlp` (`CallHlp` recorder/injector)
- Test helpers in `test/unit/common/common.go` and `helpers/ntesting/`

### Integration tests

- Framework: `github.com/onsi/ginkgo/v2` + `github.com/onsi/gomega`
- Separate module: `test/integration/go.mod`
- Prerequisites: running HLF network + Redis
- Required env vars (defined in `helpers/ntesting/ci.go`):
  - `ROBOT_TEST_IS_INTEGRATION` — must be set (else test skips)
  - `ROBOT_TEST_HLF_PROFILE` — Fabric connection profile path
  - `ROBOT_TEST_HLF_CH_FIAT`, `ROBOT_TEST_HLF_CH_CC`, `ROBOT_TEST_HLF_CH_INDUSTRIAL`, `ROBOT_TEST_HLF_CH_NO_CC` — chaincode names
  - `ROBOT_TEST_HLF_FIAT_OWNER_KEY_BASE58CHECK`, `ROBOT_TEST_HLF_CC_OWNER_KEY_BASE58CHECK` — for swap tests
  - `ROBOT_TEST_DO_SWAPS`, `ROBOT_TEST_DO_MSWAPS` — enable swap/multiswap scenarios
- Run: `go test ./... -p 1` (serial, these are heavy)
- Each suite uses unique base ports for isolation

### Lint (CI gate)

`golangci-lint run` — ensure no lint errors before pushing. Config allows `gomega`/`ginkgo` dot-imports in test files.

## Architecture

```
main.go                  → config.GetConfig(), chrobot.CreateRobots(), HTTP server
chrobot/                 → orchestrates per-channel robot lifecycle
chcollector/             → subscribes to source channel block events
collectorbatch/          → accumulates batches from block data
hlf/                     → Fabric SDK wrappers (event client, executor, block parser)
  sdk_chcollector.go     → block event subscription with reconnection
  sdk_chexecutor.go      → batchExecute TX sending with retry/split
  parser/                → block → tx/swap/multiswap/key extraction
server/                  → stdlib net/http: /info, /metrics, /healthz, /readyz
storage/redis/           → checkpoint/offset storage with optimistic locking
helpers/nerrors/         → nested error helpers
dto/ → collectordto, executordto, parserdto, stordto (data models)
```

- **Scaling:** Single or multi-instance; Redis optimistic locking prevents conflicts.
- **Metrics:** Prometheus via `github.com/prometheus/client_golang` (18 metrics, pre-initialized in `main.go:setInitMetricsVals()`).

## Release

Pushed to Docker Hub as `scientificideas/robot` (multi-arch `linux/amd64`, `linux/arm64`) on tag `v*` via `.github/workflows/release.yml`.

## Known quirks

- **License:** LICENSE file says MIT; README badge claims Apache-2.0 (MIT is the correct one).
- **SDK gap:** ChCollector handles an off-by-one block subscription gap (SDK issue #89) — logic in `hlf/sdk_chcollector.go`.
- **Large batches:** `hlf/sdk_execwithsplit.go` splits batches when ordering request size is exceeded.

## Coding Principles (Karpathy Skills)

Behavioral guidelines to reduce common LLM coding mistakes. These bias toward caution over speed; for trivial tasks, use judgment.

1. **Think Before Coding** — Don't assume. Don't hide confusion. Surface tradeoffs.
- State assumptions explicitly; ask if uncertain.
- Present multiple interpretations rather than picking silently.
- If a simpler approach exists or something is unclear, say so and ask.

2. **Simplicity First** — Minimum code that solves the problem. Nothing speculative.
- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If 200 lines could be 50, rewrite it.

3. **Surgical Changes** — Touch only what you must. Clean up only your own mess.
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken. Match existing style.
- If you notice unrelated dead code, mention it — don't delete it.
- Remove only what YOUR changes made unused.
- Every changed line should trace directly to the request.

4. **Goal-Driven Execution** — Define success criteria. Loop until verified.
- "Add validation" → "Write tests for invalid inputs, then make them pass".
- "Fix the bug" → "Write a test that reproduces it, then make it pass".
- "Refactor X" → "Ensure tests pass before and after".
- For multi-step tasks, state a brief plan with `verify:` checks per step.
