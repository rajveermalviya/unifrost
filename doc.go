// Copyright 2019-2020 Rajesh Malviya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
