WMS reverse proxy for 3di scalability
=====================================

On the scalability staging and production machines we have a WMS reverse proxy
that is based on nginx and lua.

For local development, you can use an executable based on the 
wms_reverse_proxy.go file.

Instructions for local deployment
---------------------------------

Make sure you have the latest version of Go 1.x installed. Then build an
executable, like this::

    $ go build -o wmsrp .

To see the command line options of the generated executable, do::

    $ ./wmsrp -h

Or to run it, do (values are examples)::

    $ /.wmsrp -proxy_port=8321 -redis_ip=10.0.3.100

You can determine the proxy_port yourself, but it has to match the wms url 
port in your threedi_server settings. For example::

    THREEDI_WMS_SERVER_URL = 'http://localhost:8321/3di/wms'
    THREEDI_WMS_DATA_URL = 'http://localhost:8321/3di/data'

The redis_ip value must point to your threedi_server redis server.

When you running the wms reverse proxy with the proper settings as stated 
above, your wms results will be served through the reverse proxy.

