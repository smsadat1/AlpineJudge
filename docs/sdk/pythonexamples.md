```python
import alpinejudge as aj

client = aj.Client("https://aj.example.com")

job = client.submit_file(
    language="cpp",
    version="c++17",
    file="main.cpp",
    testset="t001",
    testset_version="v1",
)

for event in job.events():
    print(event)

result = job.wait()

print(result.verdict)
print(result.stdout)
print(result.stderr)
print(result.exec_time_ms)
```

Alternative usage:

```python
job = client.submit_code(
    language="python",
    version="python3.12",
    source_code="""
print("Hello AlpineJudge")
""",
    testset="hello",
    testset_version="v1",
)

print(job.wait().verdict)
```

Asynchronous polling:

```python
while not job.done:
    print(job.status)
    time.sleep(1)

print(job.result)
```

Result object:

```python
result.verdict
result.exec_time_ms
result.memory_kb
result.stdout
result.stderr
result.compile_stdout
result.compile_stderr
result.score
```

```
```
