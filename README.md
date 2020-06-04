# Patroni Exporter

Connects to Patroni's metrics endpoint and exports metrics in Prometheus format.

Expects Patroni to be running on localhost:8008.

Exports metrics on port 9394 at `/metrics`.

## Running Locally

You need to install Go `brew install golang`.

Then run `go build` to build and install dependencies.

To run the project run `./patroni-exporter`. Remember to run `go build` after making changes
