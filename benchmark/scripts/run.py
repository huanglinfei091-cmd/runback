#!/usr/bin/env python3
"""Run copied real evidence through the production CLI. No GitHub credentials needed."""
import argparse, json, pathlib, subprocess, time, shutil, datetime

CATEGORIES = ["SUCCESS","OUT_OF_SCOPE","RUNNER_UNSUPPORTED","SECRETS_REQUIRED","NETWORK_DEPENDENCY","ENVIRONMENT_MISMATCH","ACT_INCOMPATIBILITY","MATRIX_RESOLUTION_FAILED","WORKFLOW_UNSUPPORTED","FLAKY_TEST","FAILURE_NOT_MATCHED","UNKNOWN"]
def classify(message):
    for name in CATEGORIES[1:]:
        if name in message:
            return name
    if "candidates (requires exactly one)" in message or "matrix" in message.lower():
        return "MATRIX_RESOLUTION_FAILED"
    if "no uniquely resolvable" in message or "no failed step" in message:
        return "WORKFLOW_UNSUPPORTED"
    return "UNKNOWN"

def main():
    p=argparse.ArgumentParser()
    p.add_argument("--evidence",required=True)
    p.add_argument("--binary",default="./bin/runback")
    p.add_argument("--output",default="benchmark/results/latest.json")
    p.add_argument("--workspace", default="/tmp/runback-benchmark")
    p.add_argument("--execute",action="store_true")
    p.add_argument("--timeout",default="3m")
    a=p.parse_args()
    root=pathlib.Path(a.workspace).resolve();root.mkdir(parents=True,exist_ok=True)
    output=pathlib.Path(a.output);output.parent.mkdir(parents=True,exist_ok=True)
    binary=str(pathlib.Path(a.binary).resolve())
    results=[]
    for source in sorted(pathlib.Path(a.evidence).glob("*.json")):
        started=time.monotonic()
        try:
            data=json.loads(source.read_text(encoding="utf-8-sig"))
            if "run" not in data: continue
            case=root/str(data["run"]["id"]);case.mkdir(exist_ok=True)
            lock=case/"runback.lock"
            row={"url":data["url"],"repository":data["run"]["repository"]["full_name"],
                 "run_id":data["run"]["id"],"source":"real-github-run","stage":"inspect",
                 "status":"UNKNOWN","eligible":None,"executed":False}
            cmd=[binary,"inspect",data["url"],"--bundle",str(source.resolve()),"--lock",str(lock)]
            proc=subprocess.run(cmd,capture_output=True,text=True,timeout=60)
            (case/"inspect.log").write_text(proc.stdout+proc.stderr)
            if proc.returncode:
                row["status"]=classify(proc.stdout+proc.stderr)
                row["reason"]=(proc.stdout+proc.stderr)[-2000:]
                if row["status"] in ("OUT_OF_SCOPE","RUNNER_UNSUPPORTED"):row["eligible"]=False
            else:
                plan=json.loads(lock.read_text())
                row.update(job=plan["job_name"],matrix=plan["matrix"],commit=plan["commit"])
                if plan["blockers"]:
                    row["status"]=classify("; ".join(plan["blockers"]))
                    row["reason"]="; ".join(plan["blockers"])
                    row["eligible"]=False if row["status"] in ("OUT_OF_SCOPE","RUNNER_UNSUPPORTED","SECRETS_REQUIRED") else None
                elif a.execute:
                    row["eligible"]=True
                    if shutil.disk_usage(root).free < 2*1024**3:
                        row["status"]="ENVIRONMENT_MISMATCH";row["reason"]="less than 2 GiB free space"
                    else:
                        row["executed"]=True;row["stage"]="replay"
                        proc=subprocess.run([binary,"replay","--lock",str(lock),"--work-dir",str(case),"--timeout",a.timeout],capture_output=True,text=True,timeout=1200)
                        (case/"replay.log").write_text(proc.stdout+proc.stderr)
                        reports=list(case.glob("runback-*/result.json"))
                        if reports:
                            result=json.loads(reports[0].read_text())
                            row["status"]=result["status"];row["result"]=result
                        else:
                            row["status"]=classify(proc.stdout+proc.stderr);row["reason"]=(proc.stdout+proc.stderr)[-2000:]
                else:
                    row["eligible"]=True;row["reason"]="plan resolved; replay not attempted"
        except Exception as exc:
            row={"source":str(source),"status":"UNKNOWN","eligible":None,"executed":False,"reason":str(exc)}
        row["duration_seconds"]=round(time.monotonic()-started,3)
        results.append(row)
        summary={"collected":len(results),"outside_scope":sum(x.get("eligible") is False for x in results),
                 "eligible":sum(x.get("eligible") is True for x in results),
                 "unclassified":sum(x.get("eligible") is None for x in results),
                 "executed":sum(x.get("executed",False) for x in results),
                 "reproduced":sum(x["status"]=="SUCCESS" for x in results)}
        # No extrapolation from collected/inspected cases into execution success rate.
        summary["eligible_reproduction_rate"]=summary["reproduced"]/summary["eligible"] if summary["eligible"] and a.execute else None
        output.write_text(json.dumps({"measured_at":datetime.datetime.now(datetime.timezone.utc).isoformat(),"mode":"execute" if a.execute else "inspect","summary":summary,"cases":results},indent=2)+"\n")
        print(row.get("url",source),row["status"],flush=True)
    print(json.dumps(summary,indent=2))

if __name__=="__main__":main()
