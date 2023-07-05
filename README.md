# Webhook logger

> A simple webhook logger

The aim of this go program is to aid debugging webhooks.

You can deploy it on [koyeb][k] very easily (although it is not specific for
that platform).

Once deployed, you can point your webhook towards the `/webhook` path.
Connecting with a browser on the base url, you will be served a simple html
page that listens for events from a websocket.

All values are optional. If provided at startup, a valid `Bearer` token in the
`Authorization` header must be provided with each webhook.

### Usage

```
NAME:
   webhook-logger - A simple webhook logger

USAGE:
   webhook-logger [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug         set log level to debug (default: false) [$DEBUG]
   --port value    port to listen on (default: 8080) [$PORT]
   --bearer value  bearer token to authenticate with [$BEARER]
   --help, -h      show help

```


[k]: https://www.koyeb.com
