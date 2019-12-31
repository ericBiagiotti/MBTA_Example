package main

// Struct to marshal individual route data
type routeData struct {
	Attributes struct {
		LongName string `json:"long_name"`
	} `json:"attributes"`
	ID string `json:"id"`
}

// Struct to marshal route query responses
type routesResponse struct {
	Data []routeData `json:"data"`
}

// Returns a slice containing the long names from the API response
func getLongNames(routes *routesResponse) []string {
	results := make([]string, len(routes.Data))
	for i, route := range routes.Data {
		results[i] = route.Attributes.LongName
	}
	return results
}
