#!/usr/bin/env python3
import os
import sys
import time
import requests
import threading
from typing import Optional
from pydantic import BaseModel

# Configuration from environment
CORE_URL = os.getenv("CODEPOM_CORE_URL", "http://localhost:8000")
WORKER_ID = os.getenv("WORKER_ID", f"worker-{os.uname().nodename}")
POLL_INTERVAL = int(os.getenv("POLL_INTERVAL", "5"))

class Job(BaseModel):
    id: int
    tenantId: str
    jobType: str
    payload: dict
    attempts: int
    leaseToken: str

def start_heartbeat_thread(job_id: int, lease_token: str, stop_event: threading.Event):
    """Sends periodic heartbeats to Core to retain job lease ownership."""
    def run():
        url = f"{CORE_URL}/api/v1/jobs/heartbeat"
        payload = {
            "job_id": job_id,
            "worker_id": WORKER_ID,
            "lease_token": lease_token
        }
        while not stop_event.is_set():
            time.sleep(30)
            if stop_event.is_set():
                break
            try:
                res = requests.post(url, json=payload, timeout=5)
                if res.status_code == 409:
                    print(f"⚠️ [Heartbeat] Lease conflict on Job #{job_id}. Preempted by reaper.", file=sys.stderr)
                    stop_event.set()
                elif res.status_code != 200:
                    print(f"⚠️ [Heartbeat] Unexpected status {res.status_code} for Job #{job_id}.", file=sys.stderr)
            except Exception as e:
                print(f"⚠️ [Heartbeat] Request failed for Job #{job_id}: {e}", file=sys.stderr)

    t = threading.Thread(target=run, daemon=True)
    t.start()

def execute_job(job: Job) -> dict:
    """Executes the specific LLM consensus or triage job."""
    print(f"🐾 Executing job {job.id} (Type: {job.jobType}) for Tenant: {job.tenantId}...")
    
    # Simulate LLM consensus run
    time.sleep(10)
    
    return {
        "status": "completed",
        "result": {
            "is_valid": True,
            "reasoning": f"Successfully triaged issue {job.payload.get('issue_key')} on rule {job.payload.get('rule_key')}."
        }
    }

def main():
    print(f"🐾 Starting CodePom Worker: {WORKER_ID}")
    print(f"🐾 CodePom Core Host: {CORE_URL}")

    while True:
        try:
            # Poll Core server to claim a job
            url = f"{CORE_URL}/api/v1/jobs/claim"
            res = requests.post(url, json={"worker_id": WORKER_ID}, timeout=10)
            
            if res.status_code == 204:
                # No jobs pending, sleep and poll again
                time.sleep(POLL_INTERVAL)
                continue
            
            if res.status_code != 200:
                print(f"⚠️ Failed to claim jobs. Server returned status: {res.status_code}", file=sys.stderr)
                time.sleep(POLL_INTERVAL)
                continue

            # Process the claimed job
            job_data = res.json()
            job = Job(**job_data)
            
            stop_event = threading.Event()
            start_heartbeat_thread(job.id, job.leaseToken, stop_event)

            try:
                # Run the execution
                result = execute_job(job)
                
                # Check if the heartbeat thread detected preemption
                if stop_event.is_set():
                    print(f"❌ Aborted completing Job #{job.id} because the lease was preempted.", file=sys.stderr)
                    continue

                # Phase 3: Complete the job with optimistic locking
                comp_url = f"{CORE_URL}/api/v1/jobs/complete"
                comp_payload = {
                    "job_id": job.id,
                    "worker_id": WORKER_ID,
                    "lease_token": job.leaseToken,
                    "attempts": job.attempts
                }
                comp_res = requests.post(comp_url, json=comp_payload, timeout=5)
                if comp_res.status_code == 200:
                    print(f"✅ Job #{job.id} successfully marked as completed.")
                else:
                    print(f"❌ Failed to complete Job #{job.id}. Status: {comp_res.status_code}", file=sys.stderr)
            except Exception as e:
                # Fail the job
                print(f"❌ Job #{job.id} failed with error: {e}", file=sys.stderr)
                if not stop_event.is_set():
                    fail_url = f"{CORE_URL}/api/v1/jobs/fail"
                    fail_payload = {
                        "job_id": job.id,
                        "worker_id": WORKER_ID,
                        "lease_token": job.leaseToken,
                        "attempts": job.attempts,
                        "error_message": str(e)
                    }
                    requests.post(fail_url, json=fail_payload, timeout=5)
            finally:
                stop_event.set()

        except Exception as e:
            print(f"⚠️ Worker polling encountered error: {e}", file=sys.stderr)
            time.sleep(POLL_INTERVAL)

if __name__ == "__main__":
    main()
