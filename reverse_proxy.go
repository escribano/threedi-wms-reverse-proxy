package main

import (
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

func wmsReverseProxy(redisHost string, redisPort string, useCache bool, singleServer string) *httputil.ReverseProxy {
	// normally use redis to find subgrid_id and ip belonging to session
	// if singleServer is defined, subgrid_id = singleServer and redirect to localhost
	var subgridID string

	sessionKeyCache := make(map[string]string)

	redisAddress := strings.Join([]string{redisHost, redisPort}, ":")
	pool = newRedisPool(redisAddress, "")

	director := func(req *http.Request) {
		var startTime = time.Now()
		log.Println("- INFO - start resolving wms address")
		// 1) get the session key
		var sessionKey, addressKey, dataAddress string

		sessionCookie, err := req.Cookie("sessionid")
		remoteAddr := strings.Split(req.RemoteAddr, ":")[0] // skip the port
		if err == nil {
			sessionKey = sessionCookie.Value
			log.Println("- INFO - got session key from request:", sessionKey)
			if useCache == true {
				// put the session key in a cache map by using remoteAddr as key
				sessionKeyCache[remoteAddr] = sessionKey
				log.Println("- INFO - storing session key in cache; remote address:", remoteAddr)
			}
		} else if useCache == true {
			log.Println("- INFO - fetching session key from cache; remote address:", remoteAddr)
			sessionKey = sessionKeyCache[remoteAddr]
			if sessionKey != "" {
				log.Println("- INFO - got session key from cache:", sessionKey)
			}
		}
		if sessionKey == "" {
			log.Println("- ERROR - unable to get a session key")
			req.URL = nil // results in a 500 response, which is what we want here
			return
		}

		// redis connection from pool
		// TODO: how to define conn: *redis.pooledConnection ??
		conn := pool.Get()
		log.Println("- DEBUG - number of active connections in redis pool:", pool.ActiveCount())
		defer conn.Close()

		// 2) get the subgrid_id
		if singleServer == "" {
			subgridID, err = redis.String(conn.Do("HGET", "session_to_subgrid_id", sessionKey))
			if err != nil {
				log.Println("- ERROR - unable to get subgrid id:", err)
				req.URL = nil
				return
			}
			log.Println("- INFO - got subgrid id:", subgridID)
		} else {
			subgridID = singleServer
			log.Println("- INFO - single server mode, subgrid_id:", subgridID)
		}

		// 3) get the loaded_model_type
		loadedModelType, err := redis.String(conn.Do("GET", subgridID+":loaded_model_type"))
		if err != nil {
			// assume loaded_model_type is 3di for backwards compatibility
			log.Println("- WARNING - loaded_model_type not available, fallback to default (3di)")
			loadedModelType = "3di"
		}
		log.Println("- INFO - using loaded model type:", loadedModelType)

		// 4) determine which address key to use based on the loaded model type
		switch loadedModelType {
		case "3di":
			addressKey = "wms_address"
		case "3di-urban":
			addressKey = "flow_address"
		default:
			log.Println("- ERROR - unsupported loaded model type:", loadedModelType)
			req.URL = nil
			return
		}

		// 5) get the data address
		if singleServer == "" {
			dataAddress, err = redis.String(conn.Do("HGET", "subgrid_id_to_"+addressKey, subgridID))
			if err != nil {
				log.Println("- ERROR - unable to get data address:", err)
				req.URL = nil
				return
			}
		} else {
			switch loadedModelType {
			case "3di":
				// 5000 is the default wms port
				dataAddress = "127.0.0.1:5000"
			case "3di-urban":
				// 6000 is the default flow port
				dataAddress = "127.0.0.1:6000"
			default:
				log.Println("- ERROR - unsupported loaded model type:", loadedModelType)
				req.URL = nil
				return
			}
		}

		// redirect this request to the resolved data address
		var endTime = time.Now()
		log.Println("- INFO - resolved data address", dataAddress, "in", endTime.Sub(startTime))
		req.URL.Scheme = "http"
		req.URL.Host = dataAddress
	}
	return &httputil.ReverseProxy{Director: director}
}
