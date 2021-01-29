## Sources SuperKey Worker

This is the worker that will run and process the superkey creation steps. 

#### To build:
`go build` 

#### Layout
```
amazon/       - aws api client files for s3, iam, etc
config/       - config struct handling
logger/       - <self explanatory>
messaging/    - kafka client 
sources/      - sources api client wrapper
util/         - various utilities for producing messages etc

Containerfile - container spec with builder image, based on UBI
main.go       - how to GO!
```