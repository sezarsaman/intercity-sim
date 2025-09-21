package mq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	ExchangeEvents = "rides.events"

	// Main queues
	QueueTripRequested = "pricing.q.trip_requested"
	QueueTripPriced    = "trip.q.trip_priced"

	// Retry & DLQ
	QueueTripRequestedR  = "trip.q.trip_requested.retry"
	QueueTripRequestedDL = "trip.q.trip_requested.dlq"

	QueueTripPricedR  = "trip.q.trip_priced.retry"
	QueueTripPricedDL = "trip.q.trip_priced.dlq"

	// Routing keys
	rkTripRequested      = "trip.requested"
	rkTripRequestedRetry = "trip.requested.retry"
	rkTripRequestedDLQ   = "trip.requested.dlq"

	rkTripPriced      = "trip.priced"
	rkTripPricedRetry = "trip.priced.retry"
	rkTripPricedDLQ   = "trip.priced.dlq"
)

func BootstrapTopology(amqpURL string) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return EnsureTopology(ch)
}

func EnsureTopology(ch *amqp.Channel) error {
	// exchange
	if err := ch.ExchangeDeclare(ExchangeEvents, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	// helpers
	bind := func(q, key string) error {
		return ch.QueueBind(q, key, ExchangeEvents, false, nil)
	}
	declare := func(name string, args amqp.Table) error {
		_, err := ch.QueueDeclare(name, true, false, false, false, args)
		return err
	}

	// --- trip.requested flow (publisher → pricing) ---

	// main queue: bind to event key; DLX → retry key
	if err := declare(QueueTripRequested, amqp.Table{
		"x-dead-letter-exchange":    ExchangeEvents,
		"x-dead-letter-routing-key": rkTripRequestedRetry,
	}); err != nil {
		return err
	}
	if err := bind(QueueTripRequested, rkTripRequested); err != nil {
		return err
	}

	// retry queue: TTL; DLX → original event key; **bind only to retry key**
	if err := declare(QueueTripRequestedR, amqp.Table{
		"x-message-ttl":             int32(15000),
		"x-dead-letter-exchange":    ExchangeEvents,
		"x-dead-letter-routing-key": rkTripRequested,
	}); err != nil {
		return err
	}
	if err := bind(QueueTripRequestedR, rkTripRequestedRetry); err != nil {
		return err
	}

	// DLQ: bind only to DLQ key
	if err := declare(QueueTripRequestedDL, nil); err != nil {
		return err
	}
	if err := bind(QueueTripRequestedDL, rkTripRequestedDLQ); err != nil {
		return err
	}

	// --- trip.priced flow (publisher → trip-service) ---

	// main
	if err := declare(QueueTripPriced, amqp.Table{
		"x-dead-letter-exchange":    ExchangeEvents,
		"x-dead-letter-routing-key": rkTripPricedRetry,
	}); err != nil {
		return err
	}
	if err := bind(QueueTripPriced, rkTripPriced); err != nil {
		return err
	}

	// retry (TTL back to main)
	if err := declare(QueueTripPricedR, amqp.Table{
		"x-message-ttl":             int32(15000),
		"x-dead-letter-exchange":    ExchangeEvents,
		"x-dead-letter-routing-key": rkTripPriced,
	}); err != nil {
		return err
	}
	if err := bind(QueueTripPricedR, rkTripPricedRetry); err != nil {
		return err
	}

	// DLQ
	if err := declare(QueueTripPricedDL, nil); err != nil {
		return err
	}
	if err := bind(QueueTripPricedDL, rkTripPricedDLQ); err != nil {
		return err
	}

	return nil
}
