package mq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	ExchangeEvents = "rides.events"

	QueueTripRequested   = "trip.q.trip_requested"
	QueueTripRequestedR  = "trip.q.trip_requested.retry"
	QueueTripRequestedDL = "trip.q.trip_requested.dlq"

	QueueTripPriced   = "trip.q.trip_priced"
	QueueTripPricedR  = "trip.q.trip_priced.retry"
	QueueTripPricedDL = "trip.q.trip_priced.dlq"

	rkTripRequested = "trip.requested"
	rkTripPriced    = "trip.priced"
)

// DeclareTopology declares exchanges/queues/bindings including retry & DLQ.
// Uses classic-TTL-queue pattern (no plugins required).
func DeclareTopology(ch *amqp.Channel) error {
	// exchange
	if err := ch.ExchangeDeclare(ExchangeEvents, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	// helper
	declare := func(queue, deadLetter string, ttl int32, bindKey string) error {
		args := amqp.Table{}
		if deadLetter != "" {
			args["x-dead-letter-exchange"] = ExchangeEvents
			args["x-dead-letter-routing-key"] = bindKey
		}
		if ttl > 0 {
			args["x-message-ttl"] = int32(ttl)
		}
		if _, err := ch.QueueDeclare(queue, true, false, false, false, args); err != nil {
			return err
		}
		return nil
	}
	bind := func(queue, key string) error {
		return ch.QueueBind(queue, key, ExchangeEvents, false, nil)
	}

	// trip.requested (→ pricing)
	if err := declare(QueueTripRequestedDL, "", 0, rkTripRequested); err != nil {
		return err
	}
	if err := declare(QueueTripRequestedR, QueueTripRequested, 15000, rkTripRequested); err != nil {
		return err
	} // 15s backoff
	if err := declare(QueueTripRequested, QueueTripRequestedR, 0, rkTripRequested); err != nil {
		return err
	}
	if err := bind(QueueTripRequested, rkTripRequested); err != nil {
		return err
	}
	if err := bind(QueueTripRequestedR, rkTripRequested); err != nil {
		return err
	}

	// trip.priced (→ trip-service)
	if err := declare(QueueTripPricedDL, "", 0, rkTripPriced); err != nil {
		return err
	}
	if err := declare(QueueTripPricedR, QueueTripPriced, 15000, rkTripPriced); err != nil {
		return err
	}
	if err := declare(QueueTripPriced, QueueTripPricedR, 0, rkTripPriced); err != nil {
		return err
	}
	if err := bind(QueueTripPriced, rkTripPriced); err != nil {
		return err
	}
	if err := bind(QueueTripPricedR, rkTripPriced); err != nil {
		return err
	}

	return nil
}

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

	return DeclareTopology(ch)
}
