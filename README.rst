wms reverse proxy for 3di scalability
=====================================

On the scalability staging and production machines we have a wms reverse proxy
that is based on nginx and lua.

For local development, you can use an executable based on this package.

instructions for local deployment
---------------------------------

Make sure you have Go 1.x installed. Then build the executable::

    $ go build -o wmsrp .

To see usage information and the command line options of the generated executable, do::

    $ ./wmsrp -h

Or to run it, do (values are examples)::

    $ /.wmsrp --port=8321 --redis-host=10.0.3.100 --wms-port=5000

You can determine the port yourself, but it has to match the wms url 
port in your threedi_server settings. For example::

    THREEDI_WMS_SERVER_URL = 'http://localhost:8321/3di/wms'
    THREEDI_WMS_DATA_URL = 'http://localhost:8321/3di/data'

The --redis-host value should point to your threedi server's redis server.

Add --use-cache for local development setup (because some wms requests do not have 
session cookies for some unknown reason).

When you running the wms reverse proxy with the proper settings as stated 
above, your wms results will be served through the reverse proxy.

