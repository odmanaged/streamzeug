/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

#include "DTAPI.h"
#include "asi.h"
#include <string>
#include <cstring>
#include <stdio.h>
#include <stdint.h>
#include <inttypes.h>
#include <mutex>

// Command-line program TsOut to transmit a TS file out of a DTA-2145
#define BUFSIZE (1 * 1024 * 1024)
// 64kB buffer size

#define INITIAL_LOAD (4*1024*1024) // 4MB initial load
#define LOWMARK (128 * 1024)//We should never reach this, possibly our source was lost

#define STATE_FAILED -1
#define STATE_PREFILL 0
#define STATE_RUNNING 1

#undef dektec_asi_ctx_t

struct DektecAsiCtx {
private:
    log_cb_func_t m_logcb;
    void * m_logcb_cookie;

    int m_asiportno;
    int m_bitrate;
    bool m_bidirectional = false;

    DtDevice m_device;
    DtOutpChannel m_output;


    int m_state = STATE_PREFILL;
    char m_buf[BUFSIZE] = {0};
    int bytesRemainingBuffer;
    int bytesRemainingPrefill;
    size_t bufferOffset;
    size_t bytesWritten;
    size_t bytesSinceStatsCall;
    std::mutex ctxLock;

    void writeDevice() {
        m_output.Write(m_buf, BUFSIZE, 0);
    }
    void setupPreRoll() {
        m_output.ClearFifo();
        m_output.SetTxControl(DTAPI_TXCTRL_HOLD);
        bytesRemainingBuffer = BUFSIZE;
        bytesRemainingPrefill = INITIAL_LOAD;
	m_state = STATE_PREFILL;
        bufferOffset = 0;
        bytesWritten = 0;
    }
    void setupDektecDevice() {
        char msg[1024] = {'\0'};

        DtHwFuncDesc HwFuncs[10] {};
        int f, NumberOfHwFuncs;
        ::DtapiHwFuncScan(10, NumberOfHwFuncs, HwFuncs);

        for (f=0; f<NumberOfHwFuncs; f++) {
            if (HwFuncs[f].m_Port == m_asiportno)
                break;
        }

        if (f == NumberOfHwFuncs) {
            m_logcb(m_logcb_cookie, 1, "No DekTec device found, aborting");
            m_state = STATE_FAILED;
            return;
        }

        char type_loc[128];
        ::DtapiDtHwFuncDesc2String(&HwFuncs[f], DTAPI_HWF2STR_TYPE_AND_LOC, type_loc, 128);
        char serial[128];
        ::DtapiDtHwFuncDesc2String(&HwFuncs[f], DTAPI_HWF2STR_SN, serial, 128);
        snprintf(msg, 1024, "Found DekTec %s %s", type_loc, serial);
        m_logcb(m_logcb_cookie, 0, msg);

        if ((HwFuncs[f].m_Flags & DTAPI_CAP_ASI) == 0) {
            m_logcb(m_logcb_cookie, 1, "Port does not support ASI");
            m_state = STATE_FAILED;
            return;

        }
        if ((HwFuncs[f].m_Flags & DTAPI_CAP_OUTPUT) == 0) {
            m_logcb(m_logcb_cookie, 1, "Port is not capable of output");
            m_state = STATE_FAILED;
            return;

        }

        int bidirectional = 0;
        if ((HwFuncs[f].m_Flags & DTAPI_CAP_INPUT) != 0)
            bidirectional = 1;
        
        if (m_device.AttachToSerial(HwFuncs[f].m_DvcDesc.m_Serial) != DTAPI_OK) {
            m_logcb(m_logcb_cookie, 1, "Could not attach to DekTec device, aborting");
            m_state = STATE_FAILED;
            return;

        }
        m_bidirectional = bidirectional;
    }
    void attachDektecPort() {
        int ret;
        Dtapi::DtOutpChannel out;
        ret = m_device.SetIoConfig(m_asiportno, DTAPI_IOCONFIG_IOSTD, DTAPI_IOCONFIG_ASI);
        if (ret != DTAPI_OK) {
            m_logcb(m_logcb_cookie, 1, "Could not set to ASI mode");
            m_state = STATE_FAILED;
            return;
        }
        
        if (m_bidirectional) {
            ret = m_device.SetIoConfig(m_asiportno, DTAPI_IOCONFIG_IODIR, DTAPI_IOCONFIG_OUTPUT, DTAPI_IOCONFIG_OUTPUT);
            if (ret != DTAPI_OK) {
	            m_logcb(m_logcb_cookie, 1, "Could not set to output mode");
            	m_state = STATE_FAILED;
	            return;
            }
        }
        if (m_output.AttachToPort(&m_device, m_asiportno) != DTAPI_OK) {
             m_logcb(m_logcb_cookie, 1, "Can’t attach output channel.");
	     m_state = STATE_FAILED;
             return;
	}

        // Initialise bit rate and packet mode
        m_output.SetTsRateBps(m_bitrate);
        m_output.SetTxMode(DTAPI_TXMODE_188, DTAPI_TXSTUFF_MODE_ON);
    }
public:
    DektecAsiCtx(int device_port_no, int bitrate, log_cb_func_t log_cb, void * log_cb_cookie):
        m_asiportno(device_port_no),
        m_bitrate(bitrate),
        m_logcb(log_cb),
        m_logcb_cookie(log_cb_cookie)
    {
        if (m_logcb == nullptr) {
            m_state = STATE_FAILED;
            return;
        }
        setupDektecDevice();
        if (m_state == STATE_FAILED) {
            return;
        }
        attachDektecPort();
        if (m_state == STATE_FAILED) {
            return;
        }
        setupPreRoll();
    }
    ~DektecAsiCtx() {
        const std::lock_guard<std::mutex> lock(ctxLock);
        m_output.Detach(DTAPI_INSTANT_DETACH);
        m_device.Detach();
    }
    ssize_t Write(const char buf[], size_t count) {
        const std::lock_guard<std::mutex> lock(ctxLock);
        if (m_state == STATE_FAILED) {
            return -1;
        }
        //This would require calling writeDevice() twice, which we do not support.
        if (count > (bytesRemainingBuffer + BUFSIZE)) {
            return -1;
        }
        int fifobytes = 0;
        m_output.GetFifoLoad(fifobytes, 0);
        if (m_state != STATE_PREFILL && fifobytes < LOWMARK) {
            m_logcb(m_logcb_cookie, 1, "Fifo bytes under lowmark, restarting preroll");
            setupPreRoll();
        }
        size_t sourceOffset = 0;
        int sourceRemaining = count;
        size_t bytesToCopy = std::min(bytesRemainingBuffer, sourceRemaining);

        memcpy(&m_buf[bufferOffset], &buf[sourceOffset], bytesToCopy);
        bytesRemainingBuffer -= bytesToCopy;
	    bufferOffset += bytesToCopy;
        if (bytesRemainingBuffer == 0) {
            writeDevice();
            bytesRemainingBuffer = BUFSIZE;
            bufferOffset = 0;
        }
        if (m_state == STATE_PREFILL) {
            bytesRemainingPrefill -= bytesToCopy;
            if (bytesRemainingPrefill <= 0) {
                m_output.SetTxControl(DTAPI_TXCTRL_SEND);
                m_state = STATE_RUNNING;
                m_logcb(m_logcb_cookie, 0, "Done filling FIFO, starting output");
	    	m_state = STATE_RUNNING;
            }
        }
        sourceRemaining -= bytesToCopy;
        sourceOffset += bytesToCopy;
        if (sourceRemaining > 0) {
            bytesToCopy = std::min(bytesRemainingBuffer, sourceRemaining);
            memcpy(&m_buf[bufferOffset], &buf[sourceOffset], bytesToCopy);
            bufferOffset += bytesToCopy;
            bytesRemainingBuffer -= bytesToCopy;
        }
        bytesSinceStatsCall += count;
        bytesWritten += count;
        return count;
    }

    bool StateOK(){
        return m_state != STATE_FAILED;
    }

    void Stats(DektecAsiStats *stats) {
        if (stats == nullptr) {
            return;
        }
        const std::lock_guard<std::mutex> lock(ctxLock);
        stats->BytesSinceLastCall = bytesSinceStatsCall;
        stats->BytesWritten = bytesWritten;
        m_output.GetFifoLoad(stats->FifoBytes, 0);
        bytesSinceStatsCall = 0;
    }
};

struct DektecAsiCtx * wrapper(int device_port_no, int bitrate, log_cb_func_t log_cb, void * log_cb_cookie)
{
    return new DektecAsiCtx(device_port_no, bitrate, log_cb, log_cb_cookie);
}


extern "C" {
    dektec_asi_ctx_t setup_dektec_asi_output(int device_port_no, int bitrate, log_cb_func_t log_cb, void * log_cb_cookie)
    {
        if (log_cb == nullptr) {
            return nullptr;
        }
        auto dtctx = new DektecAsiCtx(device_port_no, bitrate, log_cb, log_cb_cookie);
        if (!dtctx->StateOK()) {
            delete(dtctx);
            return nullptr;
        }
        return dtctx;
    }

    ssize_t dektec_asi_write(dektec_asi_ctx_t ctx, const char buf[], size_t count) {
        return ctx->Write(buf, count);
    }

    void dektec_asi_get_stats(dektec_asi_ctx_t ctx,struct DektecAsiStats *stats) {
        return ctx->Stats(stats);
    }

    void dektec_asi_destroy(dektec_asi_ctx_t ctx) {
        delete(ctx);
    }
}
