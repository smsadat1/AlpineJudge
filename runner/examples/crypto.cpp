#include <iostream>
#include <string>
#include <openssl/hmac.h>
#include <openssl/evp.h>

const std::string SECRET_KEY = "test-key";

// Ultra-fast byte-to-hex lookup table
const char HEX_LOOKUP[] = "0123456789abcdef";

inline void print_hex(const unsigned char* hash, unsigned int len) {
    char hex_str[65]; // 32 bytes * 2 = 64 chars + 1 null terminator
    for (unsigned int i = 0; i < len; ++i) {
        hex_str[i * 2]     = HEX_LOOKUP[(hash[i] >> 4) & 0x0F];
        hex_str[i * 2 + 1] = HEX_LOOKUP[hash[i] & 0x0F];
    }
    hex_str[64] = '\n';
    
    // Direct stdout write bypasses std::cout formatting overhead
    std::cout.write(hex_str, 65);
}

int main() {
    // Maximize I/O performance
    std::ios_base::sync_with_stdio(false);
    std::cin.tie(NULL);

    int t;
    if (!(std::cin >> t)) return 0;

    std::string input_data;
    input_data.reserve(32); // Pre-allocate to prevent heap reallocations

    unsigned char hash[EVP_MAX_MD_SIZE];
    unsigned int hash_len = 0;

    // Reuse single OpenSSL context across all 2 million iterations
    HMAC_CTX* ctx = HMAC_CTX_new();

    while (t--) {
        std::cin >> input_data;

        HMAC_Init_ex(ctx, SECRET_KEY.c_str(), SECRET_KEY.length(), EVP_sha256(), NULL);
        HMAC_Update(ctx, reinterpret_cast<const unsigned char*>(input_data.c_str()), input_data.length());
        HMAC_Final(ctx, hash, &hash_len);

        print_hex(hash, hash_len);
    }

    HMAC_CTX_free(ctx);
    return 0;
}