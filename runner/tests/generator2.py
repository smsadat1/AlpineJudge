import os
from cryptography.hazmat.primitives import hmac, hashes
import secrets
import string

def generate_hmac(input_data: bytes, secret_key: bytes) -> bytes:
    """Generates an HMAC-SHA256 tag for the input data."""
    h = hmac.HMAC(secret_key, hashes.SHA256())
    h.update(input_data)
    return h.finalize()


def generate_hash_testset(output_dir="ts002"):
    """
    Generates 1999999 tests in a single .in and .out pair
        
    :param output_dir: Directory to save generated files
    """
    # Create the output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)

    key = b"test-key"
    char_pool = string.ascii_letters + string.digits

    in_path = os.path.join(output_dir, "001.in")
    out_path = os.path.join(output_dir, "001.out")

    total_tests = 1999999
    print(f'Generating {total_tests} hashes for ts002')

    with open(in_path, 'w') as in_file, open(out_path, 'w') as out_file:
        in_file.write(f"{total_tests}\n")

        for _ in range(total_tests): 
            input = "".join(secrets.choice(char_pool) for _ in range(16))
            output = generate_hmac(input_data=input.encode(), secret_key=key)
            in_file.write(f"{input}\n")
            out_file.write(f"{output.hex()}\n")

    print(f"Successfully created {total_tests} test for '{output_dir}/'!")


if __name__ == "__main__":
    generate_hash_testset()