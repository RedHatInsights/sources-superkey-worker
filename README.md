## Sources SuperKey Worker

This is the worker that will run and process the superkey creation steps. Consumes messages off of a topic and creates required resources in AWS or any other provider (currently only AWS is supported)

#### Makefile
To build:
`make` 

To build the container:
`make container`

To run locally:
`make run`

To run in the container:
`make runcontainer`

### Layout
|Folder|description|
|---|---|
|amazon/       | aws api client files for s3, iam, etc|
|config/       | config setup in a struct|
|logger/       | \<self explanatory>|
|messaging/    | kafka client |
|sources/      | sources api client wrapper|
|provider/     | interface & structs for various providers|
|util/         | various utilities for producing messages etc|
|Containerfile | container spec with builder image, based on UBI|
|worker.go     | the listener worker|
|main.go       | how to GO!|

##### General info
`struct` definitions are in `types.go` files, methods on said structs are divided into other files or in the same file if they don't belong anywhere else.
##### Detailed Layout Info

- amazon:  
    The `amazon/` folder contains the api client in `iam.go`, `s3.go` and `reporting.go`. 
    The `credentials.go` file contains methods on the Amazon Client struct to create a new AWS API Client.

- messaging: 
Currently only a couple functions: 
    - `Consumer(topic)` to return a consumer 
    - `ConsumeWithFunction(topic, func)` that takes a topic and applies a function on each message that comes through said topic. 

- provider:
The meat and potatoes of where the application creation happens, interfaces + structs are in `types.go`.
    - `forge.go` is where the provider gets instantiated based on request type
    - `amazon_provider.go` the current only implemented superkey provider. This file contains the logic to actually create the superkey request based on a request that comes through kafka.

## License

This project is available as open source under the terms of the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
