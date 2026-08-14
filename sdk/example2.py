import asyncio
from .client import AlpineJudge

async def run_single_submission(client: AlpineJudge, sub_id: str, code: str):
    print(f"[{sub_id}] Dispatching submission...")
    
    async for event in client.submit_and_watch(
        submission_id=sub_id,
        bucket="ajbucket",
        language="cpp",
        source=code,
        testset_id="ts_123"
    ):
        print(f"[{sub_id}] {event.status} -> {event.details or event.stdout}")

async def main():
    cpp_code = '#include <iostream>\nint main() { std::cout << "Hello!"; return 0; }'
    
    async with AlpineJudge() as judge:
        # Launch 3 judge requests simultaneously
        tasks = [
            run_single_submission(judge, f"sub_{i}", cpp_code)
            for i in range(1, 4)
        ]
        
        # Stream all 3 concurrently without blocking the thread
        await asyncio.gather(*tasks)

if __name__ == "__main__":
    asyncio.run(main())