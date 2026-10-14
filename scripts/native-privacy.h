/* Windows release configuration: retain native error codes and line numbers
 * while keeping the builder's source directory out of OpenSSL error records. */
#if defined(_MSC_VER)
#include <openssl/err.h>
#undef OPENSSL_PUT_ERROR
#define OPENSSL_PUT_ERROR(library, reason) \
  ERR_put_error(ERR_LIB_##library, 0, reason, "native-source", __LINE__)
#undef OPENSSL_PUT_SYSTEM_ERROR
#define OPENSSL_PUT_SYSTEM_ERROR() \
  ERR_put_error(ERR_LIB_SYS, 0, 0, "native-source", __LINE__);
#endif
