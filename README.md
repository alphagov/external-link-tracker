# External Link Tracker

A simple Go web application that counts hits on external URLs and then redirects to the destination.

The URL used looks like:

```
/g?url=http%3A%2F%2Fexample.com
```

Note that to redirect externally the URL *must* start with the protocol.

The functionality is based on a whitelist - if the URL isn't stored in the database the service will serve a 404.

There is currently no API for adding, removing, or fetching counts of hits.

## Developing or running the tracker

The service uses GNU Make to run:

- `make run` runs the service on `localhost:8080`
- `make test` runs the Go tests
- `make build` runs `go build` with the `GOPATH` corrected to include vendorised dependencies. This generates a binary file called `external-link-tracker`

## Dependencies

The service depends on MongoDB.
