import csv, json
from datetime import datetime, timedelta, timezone


SYSTEM_CONFIG = {"label": "test-pool-1", "release": "test-v1", "app": {"nodes": 1, "cpu_cores_per_node": 4, "memory_mib_per_node": 4096}, "judge": {"nodes": 1, "cpu_cores_per_node": 2, "memory_mib_per_node": 2048, "worker_pool_size": 1, "worker_memory_limit_mib": 512, "sandbox_memory_limit_mib": 1024}}


def make_run(path, run_id="healthy-r1", rate=1.0, saturated=False, repetition=1, collectors=True, system_config=SYSTEM_CONFIG, total_submissions=None):
    path.mkdir(parents=True)
    start = datetime(2026, 8, 26, tzinfo=timezone.utc)
    end = start + timedelta(seconds=40)
    run = {"schema_version":"judge-bench/v1","run_id":run_id,"mode":"sustained","state":"completed","repetition":repetition,"started_at":start.isoformat(),"ended_at":end.isoformat(),
           "repository":{"git_sha":"abc123","dirty":False},"target":{"base_url":"https://benchmark.invalid","problem_id":1,"problem_slug":"two-sum","language":"GO","expected_verdict":"ACCEPTED"},"users":{"selected":12},"workload":{"target_rate_per_second":rate,"arrival_duration_ms":30000,"window_ms":10000,"max_in_flight":10,"max_submissions":12,"warmup_count":0},
           "phases":{"load":{"started_at":start.isoformat(),"ended_at":(start+timedelta(seconds=30)).isoformat()},"drain":{"started_at":(start+timedelta(seconds=30)).isoformat(),"ended_at":end.isoformat()}}}
    if system_config is not None: run["system_config"] = system_config
    if total_submissions is not None:
        run["workload"].pop("arrival_duration_ms")
        run["workload"]["total_submissions"] = total_submissions
    (path/"run.json").write_text(json.dumps(run))
    subcols = ["run_id","phase","sequence","user_alias","intended_at","post_started_at","post_completed_at","attempted","accepted","http_status","submission_id","terminal_status","terminal_observed_at","submit_latency_ms","end_to_end_latency_ms","accepted_to_terminal_ms","completion_source","sse_failures","rate_limited","outcome","error_class"]
    rows=[]
    for i in range(12):
        at=start+timedelta(seconds=i*2); accepted=True
        # Saturation preserves correctness: later work completes during drain,
        # rather than being fabricated as a timeout/error.
        finished=(start+timedelta(seconds=31+i)) if saturated and i >= 7 else at+timedelta(milliseconds=1000+i*10)
        e2e=(finished-at).total_seconds()*1000
        rows.append([run_id,"load",i+1,f"bench-{i+1:03}",at.isoformat(),at.isoformat(),(at+timedelta(milliseconds=100)).isoformat(),"true",str(accepted).lower(),201,i+100,"ACCEPTED",finished.isoformat(),100,e2e,e2e-100,"sse_event",0,"false","terminal",""])
    with (path/"submissions.csv").open("w", newline="") as f: csv.writer(f).writerows([subcols,*rows])
    wcols=["run_id","phase","window_index","window_start","window_end","window_duration_ms","intended","attempted","accepted","completed","accepted_cumulative","completed_cumulative","client_outstanding","client_outstanding_peak","target_arrival_rate_per_sec","attempted_rate_per_sec","accepted_rate_per_sec","completion_rate_per_sec","e2e_latency_p95_ms"]
    wins=[]
    for i in range(3):
        completed=4 if not saturated else 2; outstanding=0 if not saturated else (i+1)*2
        wins.append([run_id,"load",i,(start+timedelta(seconds=i*10)).isoformat(),(start+timedelta(seconds=(i+1)*10)).isoformat(),10000,4,4,4,completed,(i+1)*4,(i+1)*completed,outstanding,outstanding,rate,.4,.4,completed/10,1050+i*100])
    wins.append([run_id,"drain",0,(start+timedelta(seconds=30)).isoformat(),end.isoformat(),10000,0,0,0,5,12,12,0,6,"",0,0,.5,1200])
    with (path/"windows.csv").open("w", newline="") as f: csv.writer(f).writerows([wcols,*wins])
    (path/"summary.json").write_text(json.dumps({"classification":"STABLE" if not saturated else "SATURATED","drain":{"duration_ms":10000}}))
    if collectors:
        with (path/"container-stats.csv").open("w", newline="") as f:
            writer=csv.writer(f); writer.writerow(["timestamp","node","container","cpu_percent","memory_bytes","memory_limit_bytes","memory_percent","pids"])
            for i in range(4):
                for c, cpu, mem in [("judge_worker",80,200*1024**2),("judge_sandbox",70,300*1024**2),("judge_kafka",20,600*1024**2)]: writer.writerow([(start+timedelta(seconds=i*10)).isoformat(),"judge",c,cpu,mem,1024**3,mem/(1024**3)*100,5])
        with (path/"kafka-lag.csv").open("w", newline="") as f:
            writer=csv.writer(f); writer.writerow(["timestamp","consumer_group","topic","partition","current_offset","log_end_offset","lag"])
            for i,lag in enumerate(([0,0,0,0] if not saturated else [0,2,4,0])): writer.writerow([(start+timedelta(seconds=i*10)).isoformat(),"judge-worker-v1","judge.submission.jobs",0,0,lag,lag])
    return path


def make_api_run(path, run_id="api-r1", rate=10.0, achieved=None, repetition=1, system_config=SYSTEM_CONFIG, failed=0, dropped=0):
    path.mkdir(parents=True)
    achieved = rate if achieved is None else achieved
    run = {"schema_version": "astracode.api-benchmark.run.v1", "benchmark_type": "api", "run_id": run_id,
           "repository": {"git_sha": "abc123", "dirty": False}, "target": {"base_url": "https://benchmark.invalid", "endpoint": "/api/v1/problems/two-sum"},
           "workload": {"requested_rps": rate, "duration": "10s", "preallocated_vus": 2, "max_vus": 4, "max_requests": 1000}, "system_config": system_config, "repetition": repetition}
    total = int(achieved * 10)
    summary = {"schema_version": "astracode.api-benchmark.summary.v1", "benchmark_type": "api", "run_id": run_id,
               "requested_rps": rate, "achieved_rps": achieved, "total_requests": total, "successful_requests": total-failed, "failed_requests": failed,
               "error_rate": failed / total if total else 0, "dropped_iterations": dropped,
               "latency_ms": {"p50": 10, "p90": 15, "p95": 20, "p99": 30, "max": 40}}
    (path / "run.json").write_text(json.dumps(run))
    (path / "summary.json").write_text(json.dumps(summary))
    return path


def make_massive_burst_run(path, run_id="burst-r1", burst_size=10, repetition=1, collectors=True, completed=True):
    """Synthetic cardinality fixture; it deliberately has no sustained rate."""
    path.mkdir(parents=True)
    start = datetime(2026, 8, 26, tzinfo=timezone.utc)
    end = start + timedelta(seconds=30)
    run = {"schema_version": "judge-bench/v1", "run_id": run_id, "mode": "burst", "benchmark_objective": "massive-burst", "state": "completed", "repetition": repetition,
           "started_at": start.isoformat(), "ended_at": end.isoformat(), "repository": {"git_sha": "abc123", "dirty": False},
           "target": {"base_url": "https://benchmark.invalid", "problem_id": 1, "problem_slug": "two-sum", "language": "GO", "expected_verdict": "ACCEPTED"},
           "users": {"configured": burst_size, "selected": burst_size, "one_submit_per_user": True},
           "workload": {"burst_size": burst_size, "window_ms": 10000, "max_in_flight": burst_size, "max_submissions": burst_size, "warmup_count": 0},
           "phases": {"load": {"started_at": start.isoformat(), "ended_at": (start + timedelta(milliseconds=250)).isoformat()}, "drain": {"started_at": (start + timedelta(milliseconds=250)).isoformat(), "ended_at": end.isoformat()}},
           "system_config": SYSTEM_CONFIG}
    (path / "run.json").write_text(json.dumps(run))
    columns = ["run_id","phase","sequence","user_alias","intended_at","post_started_at","post_completed_at","attempted","accepted","http_status","submission_id","terminal_status","terminal_observed_at","submit_latency_ms","end_to_end_latency_ms","accepted_to_terminal_ms","completion_source","sse_failures","rate_limited","outcome","error_class"]
    rows = []
    for i in range(burst_size):
        posted = start + timedelta(milliseconds=i * 10)
        accepted = posted + timedelta(milliseconds=20)
        terminal = accepted + timedelta(seconds=1)
        terminal_status = "ACCEPTED" if completed else ""
        terminal_at = terminal.isoformat() if completed else ""
        e2e = 1020 if completed else ""
        rows.append([run_id, "load", i + 1, f"bench-{i+1:03d}", start.isoformat(), posted.isoformat(), accepted.isoformat(), "true", "true", 201, i + 1, terminal_status, terminal_at, 20, e2e, 1000 if completed else "", "sse_event" if completed else "", 0, "false", "terminal" if completed else "completion_timeout", ""])
    with (path / "submissions.csv").open("w", newline="") as f:
        csv.writer(f).writerows([columns, *rows])
    windows = [[run_id, "load", 0, start.isoformat(), (start + timedelta(milliseconds=250)).isoformat(), 250, burst_size, burst_size, burst_size, burst_size if completed else 0, burst_size, burst_size if completed else 0, 0 if completed else burst_size, burst_size, "", burst_size / .25, burst_size / .25, burst_size / .25 if completed else 0, 1020]]
    wcols = ["run_id","phase","window_index","window_start","window_end","window_duration_ms","intended","attempted","accepted","completed","accepted_cumulative","completed_cumulative","client_outstanding","client_outstanding_peak","target_arrival_rate_per_sec","attempted_rate_per_sec","accepted_rate_per_sec","completion_rate_per_sec","e2e_latency_p95_ms"]
    with (path / "windows.csv").open("w", newline="") as f:
        csv.writer(f).writerows([wcols, *windows])
    summary = {"classification": "N/A", "drain": {"duration_ms": 29750}, "burst": {"massive": True, "attempted_intake_interval_ms": (burst_size - 1) * 10, "accepted_intake_interval_ms": (burst_size - 1) * 10, "attempted_throughput_per_sec": burst_size / ((burst_size - 1) * .01) if burst_size > 1 else None, "accepted_throughput_per_sec": burst_size / ((burst_size - 1) * .01) if burst_size > 1 else None, "peak_logical_in_flight": burst_size, "peak_active_observers": burst_size}}
    (path / "summary.json").write_text(json.dumps(summary))
    return path
