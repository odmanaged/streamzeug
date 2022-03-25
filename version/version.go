/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package version

var (
	ProjectVersion  string
	GitVersion      string
	CombinedVersion string
)

func init() {
	if ProjectVersion == "" {
		ProjectVersion = "0.0"
		GitVersion = "unknown"
	}
	CombinedVersion = ProjectVersion + "-" + GitVersion
}
