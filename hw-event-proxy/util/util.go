// Copyright 2021 The Cloud Native Events Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/redhat-cne/sdk-go/pkg/types"
	log "github.com/sirupsen/logrus"
)

// GetIntEnv get int value from env
func GetIntEnv(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if ret, err := strconv.Atoi(val); err == nil {
			return ret
		}
	}
	return fallback
}

// InitLogger initilaize logger
func InitLogger() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to debug
	if !ok {
		lvl = "debug"
	}
	// parse string, this is built-in feature of logrus
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	// set global log level
	log.SetLevel(ll)
}

// APIHealthCheck .. rest api should be ready before starting to consume api
func APIHealthCheck(uri *types.URI, delay time.Duration) (ok bool, err error) {
	log.Printf("checking for rest service health\n")
	for i := 0; i <= 5; i++ {
		log.Infof("health check %s ", uri.String())
		response, errResp := http.Get(uri.String())
		if errResp != nil {
			log.Warnf("try %d, return health check of the rest service for error  %v", i, errResp)
			time.Sleep(delay)
			err = errResp
			continue
		}
		if response != nil && response.StatusCode == http.StatusOK {
			response.Body.Close()
			log.Info("rest service returned healthy status")
			time.Sleep(delay)
			err = nil
			ok = true
			return
		}
		response.Body.Close()
	}
	if err != nil {
		err = fmt.Errorf("error connecting to rest api %s", err.Error())
	}
	return
}
