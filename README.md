# Ship

Ship is a go library for building efficient event driven applications.
It provides protoc plugin library for efficient proto-based event generation.

## Goals
- Easy to understand
- Fast
- Resilient


## How to use it

### Usage

To mark any message inside a proto file as an event.

1. Import the `schema/ship.proto` inside the proto file.
2. Add the `option (ship.event) = true;` inside the message definition.

Example:

```proto
syntax = "proto3";
package example.v1;

option go_package = "v1/example";

import "schema/ship.proto";

message SomethingCreated {
  // marking a message as an event will generate the required the files for it.
  option (ship.event) = true; 

  string id = 1;
  string name = 2;
  string created_at = 3;
}
```

### Installation

#### 1. Get the protoc plugin

Install `protoc-gen-ship` from the following command

```sh
go install github.com/Flahmingo-Investments/ship/protoc-gen-ship@latest
```

#### 2. Get the proto file

Get the `schema/ship.proto` file


#### 3. Compile the proto file

To compile.

```sh
protoc \
  -I . \
  --ship_out="some/path/" \
  path/to/proto/file
  
```


### Examples

You can see the examples in [example](example/) folder.

To build the example files, run the following commands

```sh
make example
```

### Development
