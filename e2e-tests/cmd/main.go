package main

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var (
	// env variables
	webhookURL        = "http://localhost:9087/webhook"
	avgMessagesPerSec = 10
	// test duration in seconds
	testDuration = 10
	// initial delay in seconds when pod starts
	initialDelay        = 10
	checkResp    string = "YES"
	withMsgField string = "YES"

	totalPerSecMsgCount uint64 = 0

	perf    string = "NO"
	dataDir        = "data/"
	wg      sync.WaitGroup
	tck     *time.Ticker
)

func main() {
	initLogger()
	envWebhookURL := os.Getenv("TEST_DEST_URL")
	if envWebhookURL != "" {
		webhookURL = envWebhookURL
	}

	envMsgPerSec := os.Getenv("MSG_PER_SEC")
	if envMsgPerSec != "" {
		avgMessagesPerSec, _ = strconv.Atoi(envMsgPerSec)
	}

	envTestDuration := os.Getenv("TEST_DURATION_SEC")
	if envTestDuration != "" {
		testDuration, _ = strconv.Atoi(envTestDuration)
	}

	envInitialDelay := os.Getenv("INITIAL_DELAY_SEC")
	if envTestDuration != "" {
		initialDelay, _ = strconv.Atoi(envInitialDelay)
	}

	envCheckResp := os.Getenv("CHECK_RESP")
	if envCheckResp != "" {
		checkResp = envCheckResp
	}

	envWithMsgField := os.Getenv("WITH_MESSAGE_FIELD")
	if envWithMsgField != "" {
		withMsgField = envWithMsgField
	}

	envPerf := os.Getenv("PERF")
	if envPerf != "" {
		perf = envPerf
	}

	if perf == "YES" {
		perfTest()
	} else {
		basicTest()
	}
}

func initLogger() {
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

func basicTest() {
	files, err := filepath.Glob(dataDir + "*.json")
	if err != nil {
		log.Fatal(err)
	}

	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")
	req.SetRequestURI(webhookURL)
	res := fasthttp.AcquireResponse()

	for _, file := range files {
		event, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof(string(event))
		req.SetBody(event)
		if err := fasthttp.Do(req, res); err != nil {
			log.Errorf("Sending error: %v", err)
		}
		time.Sleep(time.Second)
	}
	fasthttp.ReleaseRequest(req)
}

func perfTest() {

	eventTMP0100, err := os.ReadFile("data/TMP0100.json")
	if err != nil {
		log.Fatal(err)
	}

	eventTMP0100NoMsgField, err := os.ReadFile("data/TMP0100-no-msg-field.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Webhook URL: %v", webhookURL)
	log.Infof("Messages Per Second: %d", avgMessagesPerSec)
	log.Infof("Test Duration: %d seconds", testDuration)
	log.Infof("Initial Delay: %d seconds", initialDelay)
	log.Infof("CHECK_RESP: %v", checkResp)

	log.Debugf("Sleeping %d sec...", initialDelay)
	time.Sleep(time.Duration(initialDelay) * time.Second)

	// how many milliseconds one message takes
	avgMsgPeriodInMs := 1000 / avgMessagesPerSec
	log.Debugf("avgMsgPeriodInMs: %d", avgMsgPeriodInMs)
	midpoint := avgMsgPeriodInMs / 2

	log.Debugf("midpoint: %d", midpoint)

	totalSeconds := 0
	totalMsg := 0

	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")
	if withMsgField == "YES" {
		req.SetBody(eventTMP0100)
	} else if withMsgField == "NO" {
		req.SetBody(eventTMP0100NoMsgField)
	} else {
		log.Errorf("WITH_MESSAGE_FIELD=%v is not a valid value", withMsgField)
		os.Exit(1)
	}
	req.SetRequestURI(webhookURL)
	res := fasthttp.AcquireResponse()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range time.Tick(time.Second) {
			if totalSeconds >= testDuration {
				tck.Stop()
				fasthttp.ReleaseRequest(req)
				totalSeconds--
				log.Info("******** Test completed ********")
				log.Infof("Total Seconds : %d", totalSeconds)
				log.Infof("Total Msg Sent: %d", totalMsg)
				log.Infof("Ave Msg/Second: %2.2f", float64(totalMsg/totalSeconds))
				os.Exit(0)
			}
			log.Debugf("|Total message sent mps:|%2.2f|", float64(totalPerSecMsgCount))
			totalPerSecMsgCount = 0
			totalSeconds++
		}
	}()

	log.Infof("******** Test Started ********")
	// log these again for convenient of splitting logs
	log.Infof("Webhook URL: %v", webhookURL)
	log.Infof("Messages Per Second: %d", avgMessagesPerSec)
	log.Infof("Test Duration: %d seconds", testDuration)
	log.Infof("Initial Delay: %d seconds", initialDelay)
	log.Infof("CHECK_RESP: %v", checkResp)

	// 1ms ticker
	tck = time.NewTicker(time.Duration(1000*avgMsgPeriodInMs) * time.Microsecond)
	for range tck.C {
		if checkResp == "YES" {
			totalMsg++
			if err := fasthttp.Do(req, res); err != nil {
				totalMsg--
				log.Errorf("Sending error: %v", err)
			}
		} else if checkResp == "NO" {
			totalMsg++
			fasthttp.Do(req, res) //nolint: errcheck
		} else if checkResp == "MULTI_THREAD" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				totalMsg++
				if err := fasthttp.Do(req, res); err != nil {
					log.Errorf("Sending error: %v", err)
					totalMsg--
				}
			}()
		} else {
			log.Errorf("CHECK_RESP=%v is not a valid value", checkResp)
			os.Exit(1)
		}
		totalPerSecMsgCount++
	}
}
