# Webhook logger

> A simple webhook logger

The aim of this go program is to aid debugging webhooks.

You can deploy it on [koyeb][k] very easily (although it is not specific for
that platform).

Once deployed, you can point your webhook towards the `/webhook` path.
Connecting with a browser on the base url, you will be served a simple html
page that listens for events from a websocket.

### Usage

```
NAME:
   webhook-logger - A simple webhook logger

USAGE:
   webhook-logger [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug       set log level to debug (default: false) [$DEBUG]
   --port value  port to listen on (default: 8080) [$PORT]
   --help, -h    show helkoyebp
```


[k]: https://www.koyeb.com
