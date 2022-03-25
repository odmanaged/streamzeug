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
#include <stdbool.h>
*/
import "C"
import (
	"unsafe"

	gopointer "github.com/mattn/go-pointer"
)

type DektecAsiLoggingCB func(isErr bool, message string)

//export dektekAsiLoggingCallbackWrapper
func dektekAsiLoggingCallbackWrapper(ptr unsafe.Pointer, isErr C.bool, msg *C.char) {
	userCB := gopointer.Restore(ptr).(DektecAsiLoggingCB)

	go userCB(bool(isErr), C.GoString(msg))
}

func storeLoggingCB(fn DektecAsiLoggingCB) unsafe.Pointer {
	ptr := gopointer.Save(fn)
	return ptr
}

func unsetLoggingCB(ptr unsafe.Pointer) {
	gopointer.Unref(ptr)
}
