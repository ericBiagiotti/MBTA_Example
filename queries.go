package main

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Calls http GET with the given URL and decodes the resulting JSON
// with the object provided.
func getAndDecode(mbtaURL string, decoded interface{}) error {
	resp, err := http.Get(mbtaURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&decoded)
	if err != nil {
		return err
	}
	return nil
}

// Queries the MBTA API for the subway routes
func querySubwayRoutes(mbtaURL url.URL) (routesResponse, error) {
	mbtaURL.Path = "routes"
	mbtaURL.RawQuery += "&filter[type]=0,1&fields[route]=long_name"

	var routes routesResponse
	err := getAndDecode(mbtaURL.String(), &routes)
	if err != nil {
		return routes, err
	}
	return routes, nil
}

// Queries the MBTA API for the subway stops by route ID
func queryRouteStops(mbtaURL url.URL, routeID string) (stopsResponse, error) {
	mbtaURL.Path = "stops"
	mbtaURL.RawQuery += "&filter[route]=" + routeID + "&fields[stop]=name"

	var stops stopsResponse
	err := getAndDecode(mbtaURL.String(), &stops)
	if err != nil {
		return stops, err
	}
	return stops, nil
}
