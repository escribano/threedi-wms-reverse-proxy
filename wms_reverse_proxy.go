package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/garyburd/redigo/redis"
)

var (
	sessionKeyCache map[string]string
	pool            *redis.Pool
)

func newRedisPool(server, password string) *redis.Pool {
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

func wmsReverseProxy(redisHost string, redisPort string, wmsPort string, useCache bool) *httputil.ReverseProxy {
	sessionKeyCache := make(map[string]string)

	redisAddress := strings.Join([]string{redisHost, redisPort}, ":")
	pool = newRedisPool(redisAddress, "")

	director := func(req *http.Request) {
		// 1) get the session key
		var sessionKey string
		sessionCookie, err := req.Cookie("sessionid")
		remoteAddr := strings.Split(req.RemoteAddr, ":")[0] // skip the port
		if err == nil {
			sessionKey = sessionCookie.Value
			log.Println("got session key from request:", sessionKey)
			if useCache == true {
				// put the session key in a cache map by using remoteAddr as key
				sessionKeyCache[remoteAddr] = sessionKey
				log.Println("storing session key in cache; remote address:", remoteAddr)
			}
		} else if useCache == true {
			log.Println("fetching session key from cache; remote address: ", remoteAddr)
			sessionKey = sessionKeyCache[remoteAddr]
			if sessionKey != "" {
				log.Println("got session key from cache:", sessionKey)
			}
		}
		if sessionKey == "" {
			log.Println("unable to get a session key")
			req.URL = nil // results in a 500 response, which is what we want here
			return
		}

		// redis connection from pool
		conn := pool.Get()
		log.Println("number of active connections in redis pool:", pool.ActiveCount())
		defer conn.Close()

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
	app := cli.NewApp()
	app.Name = "wmsrp"
	app.Usage = "wms reverse proxy server for the 3di scalability architecture"
	app.Version = "0.0.1"
	app.Authors = []cli.Author{
		cli.Author{Name: "Sander Smits", Email: ""},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "port, p",
			Value: "5050",
			Usage: "port this reverse proxy serves on",
		},
		cli.StringFlag{
			Name:  "redis-host",
			Value: "127.0.0.1",
			Usage: "redis server host",
		},
		cli.StringFlag{
			Name:  "redis-port",
			Value: "6379",
			Usage: "redis server port",
		},
		cli.StringFlag{
			Name:  "wms-port",
			Value: "5000",
			Usage: "wms server port",
		},
		cli.BoolFlag{
			Name:  "use-cache",
			Usage: "cache session keys (use for development environments)",
		},
	}
	app.Action = func(c *cli.Context) {
		port := c.String("port")
		redisHost := c.String("redis-host")
		redisPort := c.String("redis-port")
		wmsPort := c.String("wms-port")
		useCache := c.Bool("use-cache")

		wmsRevProxy := wmsReverseProxy(redisHost, redisPort, wmsPort, useCache)

		log.Println("starting wms reverse proxy on port", port)
		log.Println("using redirect info from redis server on", strings.Join([]string{redisHost, redisPort}, ":"))
		log.Println("redirecting to wms servers on port", wmsPort)

		err := http.ListenAndServe(":"+port, wmsRevProxy)
		if err != nil {
			log.Fatal("http.ListenAndServe:", err)
		}
	}
	app.Run(os.Args)
}
