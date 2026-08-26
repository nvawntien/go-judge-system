import csv, json
from datetime import datetime, timedelta, timezone


def make_run(path, run_id="healthy-r1", rate=1.0, saturated=False, repetition=1, collectors=True):
    path.mkdir(parents=True)
    start = datetime(2026, 8, 26, tzinfo=timezone.utc)
    end = start + timedelta(seconds=40)
    run = {"schema_version":"judge-bench/v1","run_id":run_id,"mode":"sustained","repetition":repetition,"started_at":start.isoformat(),"ended_at":end.isoformat(),
           "target":{"problem_id":1,"language":"GO"},"workload":{"target_rate_per_second":rate,"arrival_duration_ms":30000,"window_ms":10000,"max_in_flight":10},
           "phases":{"load":{"started_at":start.isoformat(),"ended_at":(start+timedelta(seconds=30)).isoformat()},"drain":{"started_at":(start+timedelta(seconds=30)).isoformat(),"ended_at":end.isoformat()}}}
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
