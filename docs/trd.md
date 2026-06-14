*Written by an `earthling` basis of `technical_spec.md`*


# Technical Specification: gosub

## Requirements

- A subscription based producer-consumer message queue service.
- Producer will publish the message to the queue, consumer will take the messages and send to subscribers webhook url
- Every topic will have a separate queue. There can be multiple topics
- Each topic have an identifier using which publisher will send messages
- Every topic have a subscriber
- Payload in each operation is type safe, mismatch in payload when posting will cause error.
- Messages sent to subscriber will have retry mechanism, exponential backoff. Crossing max retries will send the message to DLQ

### Flow Breakdown
1. **Publish Event**: The producer sends a type-safe JSON payload to `gosub` targeting a specific **Operation**.
2. **Persistence**: `gosub` persists the message to the database immediately to prevent loss.
3. **Producer Acknowledgment**: The system responds with an HTTP `202 Accepted` to the producer.
4. **Processing**: An asynchronous worker pool polls the database for pending deliveries.
5. **Subscription Resolution**: The worker looks up all active HTTP subscriptions associated with the operation name and topic.
6. **Delivery**: The worker builds an HTTP POST request, generates an HMAC signature of the payload, and sends it to the consumer URL.
7. **Resolution**:
    - **Success (2xx)**: The delivery is marked as succeeded.
    - **Failure (Non-2xx or Timeout)**: The delivery is scheduled for retry with exponential backoff.
    - **Max Retries Exceeded**: The message is moved to a Dead Letter Queue (DLQ).
---


## Low level

1. Golang for project
2. SQLite for database
3. Chi router
