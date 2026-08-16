import os
import random

def generate_test_cases(min_val=-10**9, max_val=10**9, output_dir="ts001"):
    """
    Generates 998 test cases (.in and .out files).
    
    :param min_val: Minimum integer limit
    :param max_val: Maximum integer limit
    :param output_dir: Directory to save generated files
    """
    # Create the output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)
    
    for i in range(1, 1000):
        file_id = f"{i:03d}"
        in_path = os.path.join(output_dir, f"{file_id}.in")
        out_path = os.path.join(output_dir, f"{file_id}.out")
        
        # Generate two random integers within specified range
        num1 = random.randint(min_val, max_val)
        num2 = random.randint(min_val, max_val)
        
        # Write numbers to .in file
        with open(in_path, 'w') as in_file:
            in_file.write(f"{num1} {num2}\n")
            
        # Write sum to .out file
        with open(out_path, 'w') as out_file:
            out_file.write(f"{num1 + num2}\n")

    print(f"Successfully created 998 file pairs in '{output_dir}/'!")

if __name__ == "__main__":
    # Change min_val and max_val to adjust constraints:
    # Example: numbers between -10^9 and 10^9 (64-bit integer range testing)
    generate_test_cases(min_val=-10**9, max_val=10**9)