// Package gochan is a small package for relaying pubsub messages to the web
// via SSE(EventSource).
// It is loosely based on Twitter's implementation for real-time event-streaming
// in their new web app.
//
// It uses GO CDK (gocloud.dev) for pubsub, so it supports various vendors ->
//   Google Cloud Pub/Sub
//   Amazon Simple Queueing Service (SQS) (Pending)
//   Azure Service Bus (Pending)
//   RabbitMQ
//   NATS
//   Kafka
//   In-memory (Only for testing)
//
package gochan
