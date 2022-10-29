# Pact Plugin Example - The MATT protocol

This is an example [plugin](https://github.com/pact-foundation/pact-plugins) for the [Pact](http://docs.pact.io) framework.

It implements a custom [Protocol](https://github.com/pact-foundation/pact-plugins/blob/main/docs/protocol-plugin-design.md) and [Content Matcher](https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md) for a fictional protocol - the MATT protocol.

## Use Case

The Matt protocol is a simple text-based protocol, designed for efficient communication of messages to a Matt.

MATT messages may contain any valid UTF-8 character, where the start and end of the communication must contain the word "MATT".

i.e.  `MATT<message>MATT`

When sent over TCP, messages are terminated with the newline delimeter `\n`.

## Developing the plugin

### Build and install the plugin 

The following command will build the plugin, and install into the correct plugin directory for local development:

```
make install_local
```


### Regenerating the plugin protobuf definitions

If a new protobuf definition is required, copy into the `io_pact_plugin` folder and run the following Make task:

```
make proto
```

### Prerequsites

The protoc compiler must be installed for this plugin 