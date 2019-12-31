package main

// Struct to marshal individual stop data
type stopResponseData struct {
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
	ID string `json:"id"`
}

// Struct to marshal stop query responses
type stopsResponse struct {
	Data []stopResponseData `json:"data"`
}
