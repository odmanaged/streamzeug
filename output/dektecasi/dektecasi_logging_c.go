/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package dektecasi

/*
#cgo CFLAGS: -I ${SRCDIR}/
#cgo LDFLAGS: -L${SRCDIR}/../../ -ldektec -lstdc++ -lm -ldl
#include "../../dektec/asi.h"

extern void dektekAsiLoggingCallbackWrapper(void *cookie, bool isErr, const char *message);

void dektekAsiLoggingCB(void *cookie, bool isErr, const char *message) {
	dektekAsiLoggingCallbackWrapper(cookie, isErr, message);
}
*/
import "C"
