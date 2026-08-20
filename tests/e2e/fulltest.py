import asyncio
from alpinejudge import AlpineJudge

async def main():

    client = AlpineJudge() 
    await client.upload_testset(testset_path='ts001', testset_id='ts001')

    with open("main.cpp", "r", encoding="utf-8") as file:
        file_string = file.read()

    async for event in client.submit_and_watch(
        submission_id="sub002",
        language="cpp",
        source= file_string,
        testset_id="ts001",
        memory_limit_mb=1024,
        timeout_sec=20,
        log_limit_kb=1024,
    ):
        print(f"{event.type} {event.status} {event.details}")


if __name__ == "__main__":
    asyncio.run(main())