module github.com/unifrost/examples/nats_example

go 1.14

require (
	github.com/nats-io/nats.go v1.10.0
	github.com/unifrost/unifrost v0.0.0-00010101000000-000000000000
	gocloud.dev v0.20.0
	gocloud.dev/pubsub/natspubsub v0.20.0
)

replace github.com/unifrost/unifrost => ../../
