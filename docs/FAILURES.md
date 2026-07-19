

To avoid managing states and identity and keep aligned with the goal of statelessness. S3 bucket is used as the source of truth. 

To avoid scaling problems, Runner to RabbitMQ connection is made pull based. Whenenver a Runner comes online it connects with Rabbitmq and stats executing submissions.

---

**Failure:** Runner loses network connectivity or Network partition

**Current behavior:** Rertry with exponential backoff used with jitters to reconnect with Rabbitmq while not available while also avoiding Thundering Herd problem.

---

**Failure:** Infinite loops in program or Compilation hanging

**Current behavior:** Timeout from in-container agent `time.Duration(spec.TimeoutSec)*time.Second`
Timeout with buffer at Runner level `time.Duration(rules.Timeoutsec)*time.Second + 5`

---

**Failure:** Excessive memory allocation

**Current behavior:** Used `oci.WithMemoryLimit(memoryBytes)` to limit maximum allowed processes.

---

**Failure:** Excessive stdout spam or Huge compiler/interpreter error logs

**Current behavior:** Stdout & Stderr log limit enforced from in-container agent.

---

**Failure:** Fork bombs in program.

**Current behavior:** Used `oci.WithPidsLimit(rules.PidLimit)` to limit maximum allowed processes.

---

**Failure:** Submission queue spikes (Submission storm)

**Current behavior:** Rate limiting at Dispatcher's http server as the first line of defense. 
Runner instance having it's own queue as second line of defense.
Runner's scheduler halting submission recieving based on queue length & available system resource as last line of defense.

---

**Failure:** Bad testsets.

**Current behavior:** If testset isn't stored in strict
```
testsets/
    {testset_id}/
        001/
            in.txt 
            out.txt
        002/
            ...
```
then hard fail and submission rejected.

---

**Failure:** Runner crash during execution.

**Current behavior:** `runner.service` configured with `restart: Always` so systemd respawns runner daemon within shor time.
While if there's other runner instances online , then those will take the submissions.

---

**Failure:** Zip bomb style outputs or long empty file.

**Current behavior:** Dispatcher endpoint only accepts source as string in a JSON field where 
empty string is rejected. Leaving no scope for that.

---


**Failure:** Disk exhaustion.

**Current behavior:** Runner daemon's own created artifacts are stored in /tmp only while submission artifacts are stored in S3. 
Containers are destroyed after completion of each execution.

---

