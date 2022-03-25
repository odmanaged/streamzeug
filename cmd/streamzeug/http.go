/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
)

func startHttpServer(listen string) (*http.Server, error) {
	srv := &http.Server{Addr: listen}

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := make(map[string]interface{})
		status["status"] = "OK"
		status["OK"] = true
		statuses := make(map[string]*mainloop.Status)
		flowsLock.Lock()

		for id, fh := range flows {
			statuses[id] = fh.f.Status()
			if !statuses[id].OK {
				status["status"] = "NOT-OK"
				status["OK"] = false
			}
		}
		status["flows"] = statuses
		bytes, err := json.Marshal(status)
		if err != nil {
			logging.Log.Error().Err(err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Unable to marshal to json"))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		_, _ = w.Write(bytes)
		flowsLock.Unlock()
	})
	ec := make(chan error)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			ec <- err
		}
	}()
	select {
	case err := <-ec:
		return nil, err
	case <-time.After(1 * time.Millisecond):
		return srv, nil
	}
}
