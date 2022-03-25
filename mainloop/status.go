/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package mainloop

import "time"

type Status struct {
	OK                bool      `json:"-"`
	Status            string    `json:"status"`
	LastPacketTime    time.Time `json:"lastpackettimestamp"`
	MsSinceLastPacket int       `json:"mssincelastpacket"`
	PacketCount       int       `json:"packetcount"`
	PacketsSince      int       `json:"packetssince"`
	OutputCount       int       `json:"outputcount"`
	Bitrate           int       `json:"bitrate"`
}

func (m *Mainloop) Status() *Status {
	m.statusLock.Lock()
	defer m.statusLock.Unlock()
	var status Status
	now := time.Now()
	us := now.Sub(m.lastStatusCall).Microseconds()
	status.MsSinceLastPacket = int(now.Sub(m.primaryInputStatus.lastPacketTime).Milliseconds())
	status.Bitrate = ((m.primaryInputStatus.bytesSince * 8 * 1000000) / int(us))
	status.PacketCount = m.primaryInputStatus.packetcount
	status.PacketsSince = m.primaryInputStatus.packetcountsince
	status.LastPacketTime = m.primaryInputStatus.lastPacketTime
	status.OutputCount = len(m.outputs)

	m.primaryInputStatus.bytesSince = 0
	m.primaryInputStatus.packetcountsince = 0
	m.lastStatusCall = now

	status.Status = "OK"
	status.OK = true

	return &status
}
