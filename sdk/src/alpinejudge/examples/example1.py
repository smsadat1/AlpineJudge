import asyncio
from ..client import AlpineJudge

async def main():

    client = AlpineJudge() 

    # client.create_bucket(bucket_name='ajbucket')
    # client.upload_testset(testset_path='cf86B')

    async for event in client.submit_and_watch(
        submission_id="001",
        bucket="ajbucket",
        language="cpp",
        source= '#include <iostream>\nint main() { std::cout << "Hello World!"; return 0; }',
        testset_id="cf86B",
    ):
        print(f"{event.status} -> {event.details or event.stdout}")


if __name__ == "__main__":
    asyncio.run(main())