package events

type TripRequested struct {
	TripID      string `json:"trip_id"`
	PassengerID string `json:"passenger_id"`
}

type TripPriced struct {
	TripID     string  `json:"trip_id"`
	FinalPrice int64   `json:"final_price"`
	Surge      float64 `json:"surge_multiplier"`
}
