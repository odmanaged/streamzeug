/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package dtstats

type DektecAsiStats struct {
	AsiPortno         int
	Fifobytes         int
	BytesWrittenTotal int
	BytesWritten      int
}
