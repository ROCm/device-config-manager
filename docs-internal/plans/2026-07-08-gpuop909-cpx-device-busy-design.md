# GPUOP-909: CPX partition fails "Device busy" when a user GPU workload is still running

**Date:** 2026-07-08
**Ticket:** [GPUOP-909](https://pensando.atlassian.net/browse/GPUOP-909) (Bug, v1.5.1)
**Component:** dcm / `pkg/config_manager`

## Problem

DCM changes GPU compute/memory partition mode (e.g. to `CPX_NPS1`) in response to a node label. Before partitioning it stops its own `gpuClientSystemdServices` (amd-metrics-exporter, gpuagent), but nothing waits for **user workload pods** holding `amd.com/gpu` allocations to release the GPU device files.

Applying the `amd-dcm=up:NoExecute` taint triggers Kubernetes eviction, but the full chain — API mark → SIGTERM → container exit → kubelet device release → kernel GPU release — takes far longer than DCM's ~20s service-stop window. DCM issues `amdsmi_set_gpu_compute_partition` / `amdsmi_set_gpu_memory_partition` while the GPU is still held open, and AMD SMI returns `AMDSMI_STATUS_BUSY` ("Device busy").

Observed impact (CI 31825396): node left at `state=failure` with the `NoExecute` taint, `amd.com/gpu` allocatable → 0, cascading downstream failures. In a later run (32014539) the same test *passed* only because the retry loop happened to outlast a 10.5-minute release.

## What already exists (and what the ticket gets wrong)

- The ticket's proposal #3 ("extend retry to 60–120s") **already exists**: `RetryPartition` (`gpu_config_manager.go:940`) loops for **30 minutes** at **1-minute** intervals, stopping/restarting the systemd services each pass.
- The ticket's proposal #1 ("scan `/proc/*/fd` for `/dev/kfd` handles") is **not viable as written**: the DaemonSet runs `privileged` with host `/dev` mounted but has **no `hostPID: true`**. From inside the pod's PID namespace, `/proc/*/fd` cannot see the user workload's processes — it would only see DCM's own handles. Adding `hostPID` is a privilege/visibility escalation we want to avoid.

So the real gaps are narrower than the ticket states:

1. **No gate on user-workload GPU release** — every retry during the eviction window burns a "Device busy" failure instead of waiting for the device to be free.
2. **Premature `failure` state** — `amdSMIHelper` writes `dcm.amd.com/gpu-config-profile-state=failure` on the *first* busy error (`gpu_config_manager.go:700`), even though the 30-min retry loop keeps running. This is the likely cause of the "node permanently stuck in `state=failure`" symptom.

## Design

Three coordinated changes, all within `pkg/config_manager/gpu_config_manager.go` (plus a constant). **No DaemonSet / helm-chart / privilege changes.**

### 1. Best-effort process-list gate (Approach A, degrade-gracefully)

Before issuing a `set_gpu_*_partition` call for a GPU, query the driver for processes currently using that GPU and wait for them to drain.

- New helper, mirroring the existing count-then-fill cgo pattern in `amdsmiGetProcessorHandles`:
  - `amdsmiGetGPUProcessCount(processor_handle) (int, error)` calling `amdsmi_get_gpu_process_list` (header `amdsmi.h:7801`) — first call with `list == nil` to get `max_processes`, returning the count.
- New helper `waitForGPUIdle(processor_handle, gpu_id)`:
  - Poll `amdsmiGetGPUProcessCount` on an interval up to a bounded per-GPU timeout.
  - **count == 0** → return immediately, proceed to partition (clean fast path).
  - **count > 0** → keep polling until zero or timeout; log which/how many processes are holding the GPU.
  - **API error / timeout** → log and **proceed anyway** (return, don't fail).
- Call `waitForGPUIdle` inside `amdSMIHelper`, once per GPU, immediately before the first `set_gpu_*_partition` for that GPU (only when a partition is actually needed — after the "existing == requested, skip" checks).

**Why degrade-gracefully matters:** if `amdsmi_get_gpu_process_list` under-reports on a given driver/ROCm version it returns 0, we proceed, and we are *exactly at today's behavior* — the 30-min busy-retry loop remains the backstop underneath. The gate can only remove wasted churn; it can never become a new failure source. No node-side verification is required before shipping.

### 2. Label semantics: `in-progress` until real success or real expiry

- **Busy / retryable error** (`AMDSMI_STATUS_BUSY`): do **not** write the `failure` label. Leave the `in-progress` label (already set at the top of `RetryPartition`, line 945) standing. Set `partition_failed = true` so the retry loop continues, but skip the `AddNodeLabel(..., "failure")` write on this path (the write currently at `gpu_config_manager.go:700`, and the memory-partition busy path around line 650).
- **30-min window expires while still failing** (`RetryPartition`, `gpu_config_manager.go:989-994`): this path becomes the authoritative `failure` writer — add `AddNodeLabel(nodeName, StateLabelKey, "failure")` alongside the existing `PartitionFailed` event before restarting services and returning.
- **Non-retryable errors** (invalid/unsupported profile, no sockets, bad JSON, API init failure): unchanged — these still write `failure` immediately, because the loop intentionally does not retry them.
- **Success**: unchanged (`success` at line 740).

No new label value. `dcm.amd.com/gpu-config-profile-state` keeps its three values (`in-progress` / `success` / `failure`) because it is consumed by the out-of-repo GPU Operator controller and device-plugin, which only understand those three.

### 3. Event detail via existing `PartitionRetrying`

Surface the "waiting for workload to release the GPU" detail through the existing `K8EventPartitionRetrying` event message (and the busy log line), not through the label. Events are the human/debug channel; the label is the state-machine signal.

## New constants

Add to `pkg/globals/constants.go` (near the KMM recovery timeouts):

- `GPUIdleWaitTimeout` — per-GPU max wait for processes to drain. Proposed **60s** (comfortably inside the 1-min retry cadence; the 30-min outer loop covers longer evictions across passes).
- `GPUIdleCheckInterval` — poll interval. Proposed **5s** (matches `KMMDriverRecoveryCheckInterval`).

## Error handling summary

| Situation | Partition attempt | Label | Retry loop |
|---|---|---|---|
| Processes present, drain within timeout | proceeds after drain | `in-progress` | n/a (succeeds) |
| Processes present, still there at per-GPU timeout | proceeds anyway (busy likely) | stays `in-progress` | continues (next pass) |
| `set_*_partition` returns BUSY | — | stays `in-progress` | continues |
| 30-min window expires, still busy | — | `failure` | stops |
| Invalid/unsupported profile, etc. | — | `failure` (immediate) | stops (no retry) |
| Process-list API errors / under-reports | proceeds (count 0 / error) | per outcome above | backstop unchanged |

## Testing

- **No unit tests in repo** (E2E only, gocheck, require a live cluster) — consistent with existing practice.
- **On-cluster verification (MI350X, pending hardware):** deploy a GPU workload → trigger a CPX partition → monitor DCM logs/events to confirm `waitForGPUIdle` reports the workload and the partition does **not** fail with "Device busy" → delete the workload → confirm the process count drops to zero and the partition completes → confirm the node ends at `state=success`. Also confirm that with no workload the fast path (count 0) still partitions immediately.
- **Expiry-path testing (temporary short timeout):** the 30-min `RetryPartition` window is impractical to wait out per test iteration. For testing only, build a throwaway image with the retry window shortened to ~2–3 min (the `expiration` value in `RetryPartition`, `gpu_config_manager.go:958`), then hold a workload on the GPU past that window to confirm the loop expires and writes `state=failure` + the `PartitionFailed` event. **Revert to 30 min before pushing the change** — the short timeout must not land in the committed code.
- **Regression:** confirm non-retryable failures (bad profile) still land at `state=failure` immediately.

## Out of scope

- No DaemonSet `hostPID` / mount / privilege changes.
- No changes to AINIC partition flow (`pkg/ainic_manager`), which shares the label pattern but not this bug.
- No new label value or change to the label contract consumed by the operator.
