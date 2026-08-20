import asyncio
from ..client import AlpineJudge

async def main():

    client = AlpineJudge() 
    await client.upload_testset(testset_path='ts001', testset_id='ts001')

    async for event in client.submit_and_watch(
        submission_id="sub001",
        bucket="ajbucket",
        language="cpp",
        source= '#include <iostream>\nint main() { std::cout << "Hello World!"; return 0; }',
        testset_id="ts001",
        memory_limit_mb=1024,
        timeout_sec=20,
        log_limit_kb=1024,
    ):
        print(f"{event.status} -> {event.details or event.stdout}")


if __name__ == "__main__":
    asyncio.run(main())