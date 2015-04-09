package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

var sessionKeyCache map[string]string

func WmsReverseProxy(redisHost string, redisPort string, wmsPort string) *httputil.ReverseProxy {
	sessionKeyCache := make(map[string]string)
	redisAddress := strings.Join([]string{redisHost, redisPort}, ":")
	director := func(req *http.Request) {
		// make the redis connection; consider making this a redis pool of connections
		c, err := redis.DialTimeout("tcp", redisAddress, 0, 1*time.Second, 1*time.Second)
		if err != nil {
			log.Fatal("redis.DialTimeout: ", err)
		}
		conn := c

		// get the session key from the request
		var sessionKey string
		remoteAddr := strings.Split(req.RemoteAddr, ":")[0] // skip the port
		for _, cookie := range req.Cookies() {
			if cookie.Name == "sessionid" {
				sessionKey = cookie.Value
				log.Println("session key from cookie: ", sessionKey)
				// put the session key in a cache map by using remoteAddr as key
				sessionKeyCache[remoteAddr] = sessionKey
				log.Println("storing sessionKey in cache; remoteAddr: ", remoteAddr)
			}
		}
		if sessionKey == "" {
			log.Println("remoteAddr: ", remoteAddr)
			sessionKey = sessionKeyCache[remoteAddr]
			log.Println("session key from cache: ", sessionKey)
		}
		if sessionKey == "" {
			log.Println("unable to determine a session key")
		} else {
			// get the subgrid_id from session_to_subgrid_id based on the session key
			subgridId, err := redis.String(conn.Do("HGET", "session_to_subgrid_id", sessionKey))
			if err != nil {
				log.Println("redis.Do HGET session_to_subgrid_id: ", err)
			} else {
				log.Println("subgrid_id: ", subgridId)

				// get the wms ip address from the subgrid_id_to_ip hash
				wmsIP, err := redis.String(conn.Do("HGET", "subgrid_id_to_ip", subgridId))
				if err != nil {
					log.Println("redis.Do HGET subgrid_id_to_ip: ", err)
				} else {

					// use the wms address to redirect this request
					// TODO: make wms port a command-line flag
					wmsAddress := strings.Join([]string{wmsIP, wmsPort}, ":")
					log.Println("wms address: ", wmsAddress)
					req.URL.Scheme = "http"
					req.URL.Host = wmsAddress
				}
			}
		}
		// close the redis connection
		conn.Close()
	}
	return &httputil.ReverseProxy{Director: director}
}

func main() {
	// command-line flags
	proxyPort := flag.String("proxy_port", "", "port the reverse proxy serves on (required)")
	redisIpPtr := flag.String("redis_ip", "", "ip address of the redis server (required)")
	redisPortPtr := flag.String("redis_port", "6379", "port of the redis server")
	wmsPortPtr := flag.String("wms_port", "5000", "wms server port")

	flag.Parse()
	if *proxyPort == "" {
		log.Fatal("proxy_port is required")
	}
	if *redisIpPtr == "" {
		log.Fatal("redis_ip is required")
	}

	wms_proxy := WmsReverseProxy(*redisIpPtr, *redisPortPtr, *wmsPortPtr)
	log.Println("WMS proxy started")

	// TODO: read port from command-line
	http_err := http.ListenAndServe(":"+*proxyPort, wms_proxy)
	if http_err != nil {
		log.Fatal("ListenAndServe: ", http_err)
	}
}
