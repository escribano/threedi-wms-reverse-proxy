package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

// placeholder for injecting the real version when compiling by build.sh
var Version = "dev"

func main() {
	// also use microseconds in log messages
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	app := cli.NewApp()
	app.Name = "wmsrp"
	app.Usage = "wms reverse proxy for the 3di scalability architecture"
	app.Version = Version
	app.Authors = []cli.Author{
		cli.Author{Name: "Sander Smits", Email: ""},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{ // always, required
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
			Name:  "single-server",
			Value: "",
			Usage: "say we want no redis/machine manager stuff; use provided subgrid_id (subgrid:10000) and redirect to localhost",
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
		useCache := c.Bool("use-cache")
		singleServer := c.String("single-server")

		wmsRevProxy := wmsReverseProxy(redisHost, redisPort, useCache, singleServer)

		log.Println("- INFO - starting wms reverse proxy on port", port)
		log.Println("- INFO - using redirect info from redis server on", strings.Join([]string{redisHost, redisPort}, ":"))

		err := http.ListenAndServe(":"+port, wmsRevProxy)
		if err != nil {
			log.Fatal("- ERROR - http.ListenAndServe:", err)
		}
	}
	app.Run(os.Args)
}
