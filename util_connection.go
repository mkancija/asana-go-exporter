package main

import (
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

type resourceConnection struct {
	resp   *http.Response
	body   []byte
	retry  int
	code   int
	status bool
}

func rateLimitWait(wtime int) int {
	waitms := 100

	if counter_ratelimit_fail > 1000 {
		// panic("Rate limit exceeded - must stop.")
	} else if counter_ratelimit_fail > 20 {
		waitms = 2000
	} else if counter_ratelimit_fail > 10 {
		waitms = 1000
	} else if counter_ratelimit_fail > 0 {
		waitms = (100 * counter_ratelimit_fail)
	}

	if wtime > 0 {
		waitms = wtime
	}

	return waitms
}

func getResource(url string, retry bool) resourceConnection {

	var resource resourceConnection

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resource
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	http.DefaultClient.Timeout = time.Millisecond * 10000

	// deliberate timeoout.
	if asana_timeout_limit > 0 {
		log.Println("* Timeout limit: ", asana_timeout_limit, " ms.")
		time.Sleep(time.Millisecond * time.Duration(asana_timeout_limit))
	}

	resp, err := http.DefaultClient.Do(req)

	resource.resp = resp

	if err != nil {
		log.Println("* Resource GET error: ", err)
		log.Println(resp)

		// too many sockets open?
		// wait a bit and try again once more...
		if retry {
			log.Println("......retry.......")
			time.Sleep(time.Millisecond * 1000)
			return getResource(url, false)
		}

		return resource
	}

	// Check header istruction.
	i, strerr := strconv.Atoi(resp.Header.Get("Retry-After"))
	if strerr != nil {
		resource.retry = 0
		asana_timeout_limit = 0
	} else {
		resource.retry = i * 1000
		asana_timeout_limit = int(math.Round(float64(resource.retry) / 3))
	}

	// store the response code
	resource.code = resp.StatusCode

	// If the response is OK, read the body
	if resp.StatusCode == http.StatusOK {

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resource.status = false
			log.Println("getResource: Body response read error")
			log.Fatal(err)
		}

		resource.status = true
		resource.body = bodyBytes

		resp.Body.Close()
		counter_ratelimit_fail = 0
		asana_timeout_limit = 0

	} else {
		resource.status = false
		resp.Body.Close()
	}

	return resource
}
