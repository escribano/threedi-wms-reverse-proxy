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
var pool *redis.Pool

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func wmsReverseProxy(redisHost string, redisPort string, wmsPort string) *httputil.ReverseProxy {
	sessionKeyCache := make(map[string]string)
	redisAddress := strings.Join([]string{redisHost, redisPort}, ":")
	pool = newPool(redisAddress, "")
	director := func(req *http.Request) {
		// get a redis connection from the pool
		conn := pool.Get()
		log.Println("number of active connections in redis pool:", pool.ActiveCount())
		defer conn.Close()

		// get the session key from the request
		var sessionKey string
		remoteAddr := strings.Split(req.RemoteAddr, ":")[0] // skip the port
		for _, cookie := range req.Cookies() {
			if cookie.Name == "sessionid" {
				sessionKey = cookie.Value
				log.Println("session key from cookie:", sessionKey)
				// put the session key in a cache map by using remoteAddr as key
				sessionKeyCache[remoteAddr] = sessionKey
				log.Println("storing sessionKey in cache; remoteAddr:", remoteAddr)
			}
		}
		if sessionKey == "" {
			log.Println("remoteAddr: ", remoteAddr)
			sessionKey = sessionKeyCache[remoteAddr]
			log.Println("session key from cache:", sessionKey)
		}
		if sessionKey == "" {
			log.Println("unable to determine a session key")
		} else {
			// get the subgrid_id from session_to_subgrid_id based on the session key
			subgridID, err := redis.String(conn.Do("HGET", "session_to_subgrid_id", sessionKey))
			if err != nil {
				log.Println("redis.Do HGET session_to_subgrid_id:", err)
			} else {
				log.Println("subgrid_id:", subgridID)

				// get the wms ip address from the subgrid_id_to_ip hash
				wmsIP, err := redis.String(conn.Do("HGET", "subgrid_id_to_ip", subgridID))
				if err != nil {
					log.Println("redis.Do HGET subgrid_id_to_ip:", err)
				} else {

					// use the wms address to redirect this request
					// TODO: make wms port a command-line flag
					wmsAddress := strings.Join([]string{wmsIP, wmsPort}, ":")
					log.Println("wms address:", wmsAddress)
					req.URL.Scheme = "http"
					req.URL.Host = wmsAddress
				}
			}
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func main() {
	// command-line flags
	proxyPort := flag.String("proxy_port", "", "port the reverse proxy serves on (required)")
	redisIPPtr := flag.String("redis_ip", "", "ip address of the redis server (required)")
	redisPortPtr := flag.String("redis_port", "6379", "port of the redis server")
	wmsPortPtr := flag.String("wms_port", "5000", "wms server port")

	flag.Parse()
	if *proxyPort == "" {
		log.Fatal("proxy_port is required")
	}
	if *redisIPPtr == "" {
		log.Fatal("redis_ip is required")
	}

	wmsProxy := wmsReverseProxy(*redisIPPtr, *redisPortPtr, *wmsPortPtr)
	log.Println("WMS proxy started")

	// TODO: read port from command-line
	httpErr := http.ListenAndServe(":"+*proxyPort, wmsProxy)
	if httpErr != nil {
		log.Fatal("ListenAndServe:", httpErr)
	}
}
