# Pact Plugin Example - The MATT protocol

This is an example [plugin](https://github.com/pact-foundation/pact-plugins) for the [Pact](http://docs.pact.io) framework.

It implements a custom [Protocol](https://github.com/pact-foundation/pact-plugins/blob/main/docs/protocol-plugin-design.md) and [Content Matcher](https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md) for a fictional protocol - the MATT protocol.

## Install

```
pact-plugin-cli -y install https://github.com/mefellows/pact-plugin-matt/releases/tag/v0.0.1
```

## Use Case

The MATT protocol is a simple text-based protocol, designed for efficient communication of messages to a Matt.

MATT messages are composed of basic text values, where the start and end of the communication must contain the word "MATT".

i.e.  `MATT<message>MATT`

in BNF it would be something like this:

```
<message>   ::= <delimeter> <word> <delimeter>
<delimeter> ::= "MATT"
<word>      ::= <character> | <word> <character>
<character> ::= <letter> | <number> | <symbol>
<letter>    ::= "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" | "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z"
<number>    ::= "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
<symbol>    ::=  "|" | " " | "!" | "#" | "$" | "%" | "&" | "(" | ")" | "*" | "+" | "," | "-" | "." | "/" | ":" | ";" | ">" | "=" | "<" | "?" | "@" | "[" | "\" | "]" | "^" | "_" | "`" | "{" | "}" | "~"
```

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