// Package unifrost is a go module for relaying pubsub messages to the web using SSE(Eventsource).
// It is based on Twitter's implementation for real-time event-streaming in their new web app.
//
// Supported brokers
//
//   Google Cloud Pub/Sub
//   Amazon Simple Queueing Service
//   Azure Service Bus (Pending)
//   RabbitMQ
//   NATS
//   Kafka
//   In-memory (Only for testing)
//
// For examples check https://github.com/unifrost/unifrost/tree/master/examples/
package unifrost
