package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestQueryFuncs(t *testing.T) {
	// Start a local HTTP server for testing
	i := 0
	var responses []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		rw.Write([]byte(responses[i]))
		i++
	}))
	defer server.Close()

	// Helper function for setting a new set of responses
	setResponses := func(newResponses []string) {
		responses = newResponses
		i = 0
	}

	testURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
	}

	t.Run("multi-route response success", func(t *testing.T) {
		i = 0
		setResponses([]string{`{"data":[{"attributes":{"long_name":"Red Line"}}, {"attributes":{"long_name":"Blue Line"}}]}`})
		routes, err := querySubwayRoutes(*testURL)
		if err != nil {
			t.Error(err)
		}
		if routes.Data[0].Attributes.LongName != "Red Line" {
			t.Error()
		}
		if routes.Data[1].Attributes.LongName != "Blue Line" {
			t.Error()
		}
	})

	t.Run("multi-stop response success", func(t *testing.T) {
		setResponses([]string{`{"data":[{"attributes":{"name":"stop1"}}, {"attributes":{"name":"stop2"}}]}`})
		stops, err := queryRouteStops(*testURL, "")
		if err != nil {
			t.Error(err)
		}
		if stops.Data[0].Attributes.Name != "stop1" {
			t.Error()
		}
		if stops.Data[1].Attributes.Name != "stop2" {
			t.Error()
		}
	})

	t.Run("getStopCounts", func(t *testing.T) {
		var route routeData
		route.Attributes.LongName = "Red Line"

		var routes routesResponse
		routes.Data = append(routes.Data, route)

		setResponses([]string{`{"data":[]}`})
		testGetStopCounts(*testURL, routes, 0, t)
		setResponses([]string{`{"data":[{"attributes":{"name":"stop1"}}]}`})
		testGetStopCounts(*testURL, routes, 1, t)
		setResponses([]string{`{"data":[{"attributes":{"name":"stop1"}}, {"attributes":{"name":"stop2"}}, {"attributes":{"name":"stop3"}}]}`})
		testGetStopCounts(*testURL, routes, 3, t)
	})

	var route1 routeData
	route1.Attributes.LongName = "Red Line"
	var route2 routeData
	route2.Attributes.LongName = "Orange Line"

	var routes routesResponse
	routes.Data = append(routes.Data, route1)
	routes.Data = append(routes.Data, route2)

	t.Run("getRouteByStopCount", func(t *testing.T) {
		responses := []string{`{"data":[{"attributes":{"name":"stop1"}}, {"attributes":{"name":"stop2"}}]}`,
			`{"data":[{"attributes":{"name":"stop1"}}]}`}
		setResponses(responses)
		testRouteByStopCount(*testURL, routes, "Orange Line", 1, true, t)
		setResponses(responses)
		testRouteByStopCount(*testURL, routes, "Red Line", 2, false, t)
	})

	t.Run("getStopRoutes", func(t *testing.T) {
		responses := []string{`{"data":[{"attributes":{"name":"stop1"}}, {"attributes":{"name":"stop2"}}]}`,
			`{"data":[{"attributes":{"name":"stop1"}}]}`}
		setResponses(responses)
		expectedStops := make(stopMap)
		expectedStops["stop1"] = &stopData{"stop1", []string{"Red Line", "Orange Line"}}
		expectedStops["stop2"] = &stopData{"stop2", []string{"Red Line"}}
		expectedAdjacency := make(adjacencyMap)
		expectedAdjacency["Red Line"] = []string{"Orange Line"}
		expectedAdjacency["Orange Line"] = []string{"Red Line"}
		testGetStopRoutes(*testURL, routes, expectedStops, expectedAdjacency, t)
	})
}

func testGetStopCounts(testURL url.URL, routes routesResponse, count int, t *testing.T) {
	stopCounts, err := getStopCounts(testURL, routes)
	if err != nil {
		t.Error(err)
	}
	if stopCounts["Red Line"] != count {
		t.Error()
	}
}

func testRouteByStopCount(testURL url.URL, routes routesResponse, routeByCount string, count int, fewest bool, t *testing.T) {
	r, i, err := getRouteByStopCount(testURL, routes, fewest)
	if err != nil {
		t.Error(err)
	}
	if r != routeByCount || i != count {
		t.Error()
	}
}

func testGetStopRoutes(testURL url.URL, routes routesResponse, expectedStops stopMap, expectedAdjacency adjacencyMap, t *testing.T) {
	stops, adjacencyList, err := getStopRoutes(testURL, routes, true)
	if err != nil {
		t.Error(err)
	}

	// Check that the stops map is the same
	for k, v := range expectedStops {
		if stops[k].name != v.name {
			t.Error()
		}
		for i, route := range stops[k].routes {
			if route != v.routes[i] {
				t.Error()
			}
		}
	}

	// Check that the adjacency map is the same
	for k, v := range expectedAdjacency {
		for i, adjacentRoute := range v {
			if adjacencyList[k][i] != adjacentRoute {
				t.Error()
			}
		}
	}
}

func TestGetConnectingRoutes(t *testing.T) {
	adjacentRoutes := make(map[string][]string)

	// Test same route
	fromRoutes := []string{"Red"}
	toRoutes := []string{"Red"}
	connections := getConnectingRoutes(fromRoutes, toRoutes, adjacentRoutes)
	if connections[0] != "Red" {
		t.Error()
	}

	// Test direct connection
	fromRoutes = []string{"Red"}
	toRoutes = []string{"Blue"}
	adjacentRoutes["Red"] = []string{"Blue"}
	connections = getConnectingRoutes(fromRoutes, toRoutes, adjacentRoutes)
	if connections[0] != "Red" || connections[1] != "Blue" {
		t.Error()
	}

	// Test traversal
	fromRoutes = []string{"Red"}
	toRoutes = []string{"Orange"}
	adjacentRoutes["Red"] = []string{"Green"}
	adjacentRoutes["Green"] = []string{"Blue"}
	adjacentRoutes["Blue"] = []string{"Orange"}
	connections = getConnectingRoutes(fromRoutes, toRoutes, adjacentRoutes)
	if connections[0] != "Red" || connections[1] != "Green" || connections[2] != "Blue" || connections[3] != "Orange" {
		t.Error()
	}

	// Test no route
	fromRoutes = []string{"Red"}
	toRoutes = []string{"Blue"}
	adjacentRoutes["Red"] = []string{"Green"}
	adjacentRoutes["Green"] = []string{"Orange"}
	connections = getConnectingRoutes(fromRoutes, toRoutes, adjacentRoutes)
	if len(connections) != 0 {
		t.Error()
	}
}
