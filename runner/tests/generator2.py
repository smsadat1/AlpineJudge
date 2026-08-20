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
    Generates 199999 tests in a single .in and .out pair
        
    :param output_dir: Directory to save generated files
    """
    # Create the output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)

    key = b"test-key"
    char_pool = string.ascii_letters + string.digits

    in_path = os.path.join(output_dir, "001.in")
    out_path = os.path.join(output_dir, "001.out")

    for i in range(1, 2000000):
        input = "".join(secrets.choice(char_pool) for _ in range(16))
        output = generate_hmac(input_data=input.encode(), secret_key=key)

        with open(in_path, 'a') as in_file:
            in_file.write(f"{input}\n")
                    
        with open(out_path, 'a') as out_file:
            out_file.write(f"{output}\n")

    print(f"Successfully created 199999 test for '{output_dir}/'!")


if __name__ == "__main__":
    generate_hash_testset()