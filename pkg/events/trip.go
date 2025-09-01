package events

type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

type TripRequested struct {
	TripID      string   `json:"trip_id"`
	PassengerID string   `json:"passenger_id"`
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	VehicleType string   `json:"vehicle_type"`
}

type TripPriced struct {
	TripID     string  `json:"trip_id"`
	FinalPrice float64 `json:"final_price"`
	Surge      float64 `json:"surge"`
}
