/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

#ifdef __cplusplus
extern "C"
{
#endif
    #include <unistd.h>
    #include <stdbool.h>
    struct DektecAsiStats {
        int FifoBytes;
        size_t BytesWritten;
        size_t BytesSinceLastCall;
    };
    typedef struct DektecAsiCtx* dektec_asi_ctx_t;
    typedef void (*log_cb_func_t)(void* cookie,bool error, const char *msg);
    dektec_asi_ctx_t setup_dektec_asi_output(int device_port_no, int bitrate, log_cb_func_t log_cb, void * log_cb_cookie);
    void dektec_asi_destroy(dektec_asi_ctx_t ctx);
    ssize_t dektec_asi_write(dektec_asi_ctx_t ctx, const char buf[], size_t count);
    void dektec_asi_get_stats(dektec_asi_ctx_t ctx, struct DektecAsiStats *stats);
#ifdef __cplusplus
}
#endif