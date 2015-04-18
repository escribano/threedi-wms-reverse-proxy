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

var (
	sessionKeyCache map[string]string
	pool            *redis.Pool
)

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
		// get a redis connection
		conn := pool.Get()
		log.Println("number of active connections in redis pool:", pool.ActiveCount())
		defer conn.Close()

		// 1) get the session key
		var sessionKey string
		sessionCookie, err := req.Cookie("sessionid")
		remoteAddr := strings.Split(req.RemoteAddr, ":")[0] // skip the port
		if err == nil {
			sessionKey := sessionCookie.Value
			log.Println("got session key from request:", sessionKey)
			// put the session key in a cache map by using remoteAddr as key
			sessionKeyCache[remoteAddr] = sessionKey
			log.Println("storing session key in cache; remote address:", remoteAddr)
		} else {
			log.Println("fetching session key from cache; remote address: ", remoteAddr)
			sessionKey = sessionKeyCache[remoteAddr]
			if sessionKey == "" {
				log.Println("unable to get session key from cache")
				req.URL = nil // returns a 500 response
				return
			}
			log.Println("got session key from cache:", sessionKey)
		}

		// 2) get the subgrid_id
		subgridID, err := redis.String(conn.Do("HGET", "session_to_subgrid_id", sessionKey))
		if err != nil {
			log.Println("unable to get subgrid id:", err)
			req.URL = nil
			return
		}
		log.Println("got subgrid id:", subgridID)

		// 3) get the wms server ip
		wmsIP, err := redis.String(conn.Do("HGET", "subgrid_id_to_ip", subgridID))
		if err != nil {
			log.Println("unable to get wms ip:", err)
			req.URL = nil
			return
		}
		// use the full wms address to redirect this request to
		wmsAddress := strings.Join([]string{wmsIP, wmsPort}, ":")
		log.Println("got wms address:", wmsAddress)
		req.URL.Scheme = "http"
		req.URL.Host = wmsAddress
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
		log.Fatal("-proxy_port is required")
	}

	if *redisIPPtr == "" {
		log.Fatal("-redis_ip is required")
	}

	wmsProxy := wmsReverseProxy(*redisIPPtr, *redisPortPtr, *wmsPortPtr)
	log.Println("wms proxy started")

	httpErr := http.ListenAndServe(":"+*proxyPort, wmsProxy)
	if httpErr != nil {
		log.Fatal("http.ListenAndServe:", httpErr)
	}
}
