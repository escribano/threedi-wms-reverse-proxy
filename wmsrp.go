package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

func main() {
	// also use microseconds in log messages
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	app := cli.NewApp()
	app.Name = "wmsrp"
	app.Usage = "wms reverse proxy for the 3di scalability architecture"
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
		cli.StringFlag{
			Name:  "flow-wms-port",
			Value: "6000",
			Usage: "flow wms server port",
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
		subgridWmsPort := c.String("wms-port")
		flowWmsPort := c.String("flow-wms-port")
		useCache := c.Bool("use-cache")

		wmsRevProxy := wmsReverseProxy(redisHost, redisPort, subgridWmsPort, flowWmsPort, useCache)

		log.Println("- INFO - starting wms reverse proxy on port", port)
		log.Println("- INFO - using redirect info from redis server on", strings.Join([]string{redisHost, redisPort}, ":"))
		log.Println("- INFO - redirecting subgrid wms requests to port", subgridWmsPort, "and flow wms requests to port", flowWmsPort)

		err := http.ListenAndServe(":"+port, wmsRevProxy)
		if err != nil {
			log.Fatal("- ERROR - http.ListenAndServe:", err)
		}
	}
	app.Run(os.Args)
}
