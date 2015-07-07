wms reverse proxy for 3di scalability
=====================================

instructions for local deployment
---------------------------------

Make sure you have Go 1.x installed. Then build the executable::

    $ go build -o wmsrp .

To see usage information and the command line options of the generated executable, do::

    $ ./wmsrp -h

Or to run it, do (values are examples)::

    $ /.wmsrp --port=8321 --redis-host=10.0.3.100 --wms-port=5000 --flow-wms-port=6000

You can determine the port yourself, but it has to match the wms url
port in your threedi_server settings. For example::

    THREEDI_WMS_SERVER_URL = 'http://localhost:8321/3di/wms'
    THREEDI_WMS_DATA_URL = 'http://localhost:8321/3di/data'

The --redis-host value should point to your threedi server's redis server.

Add --use-cache for local development setup (because some wms requests do not have
session cookies for some unknown reason).

When you running the wms reverse proxy with the proper settings as stated
above, your wms results will be served through the reverse proxy.

instruction for deployment on staging and production
----------------------------------------------------

Make sure supervisor is installed::

    $ apt-get install supervisor -y

Create directory /usr/local/wmsrevproxy and put the wmsrp binary in it::

    $ mkdir /usr/local/wmsrevprox
    $ cp wmsrp /usr/local/wmsrevprox

Create a /etc/supervisor/conf.d/wmsrevprox.conf file with this content (N.B. change --redis-host to p-3di-red-d1.external-nens.local for production)::

    [program:wms_reverse_proxy]
    command = /usr/local/wmsrevprox/wmsrp -p 6666 --redis-host=s-3di-red-d1.external-nens.local --wms-port=5000 --flow-wms-port=6000
    process_name = wms_reverse_proxy
    directory = /usr/local/wmsrevprox
    priority = 20
    redirect_stderr = true

Add log rotation to /etc/supervisor/supervisor.conf::

    [supervisord]
    logfile=/var/log/supervisor/supervisord.log
    pidfile=/var/run/supervisord.pid
    childlogdir=/var/log/supervisor
    stdout_logfile_maxbytes = 104857600
    stdout_logfile_backups = 10
    stderr_logfile_maxbytes = 104857600
    stderr_logfile_backups = 10

Make sure the /3di location in the nginx site config file points to the reverse proxy (proxy pass host and port must match wmsrp host and port)::

    location /3di {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $http_host;
        proxy_redirect off;
        proxy_pass http://127.0.0.1:6666;
        proxy_read_timeout 60;
    }

Reload nginx and restart supervisor::

    $ service nginx reload
    $ service supervisor restart

The wms reverse proxy log file can be found here::

    $ tail -f /var/log/supervisor/wms_reverse_proxy-stdout---supervisor-*.log

