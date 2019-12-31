package main

import (
	"container/list"
	"flag"
	"log"
	"math"
	"net/url"
	"strings"
)

// Struct to store the name of a stop and the routes that its on
type stopData struct {
	name   string
	routes []string
}

// Alias for map from stop id to stop data
type stopMap = map[string]*stopData

// Alias for map from route id to a list of adjacent routes
type adjacencyMap = map[string][]string

// Helper function for finding string list entries
func contains(list []string, value string) bool {
	for _, val := range list {
		if val == value {
			return true
		}
	}
	return false
}

// Does a breadth-first search of the routes graph until the destination route is found.
// Returns the map containing each route visited mapped to its parent route. This map can be traversed
// to find the shortest path from destRoute to srcRoute.
// Inspired by https://www.geeksforgeeks.org/shortest-path-unweighted-graph/
func buildTraversal(srcRoute string, destRoute string, adjacentRoutes adjacencyMap) map[string]string {
	parents := make(map[string]string, len(adjacentRoutes))
	queue := list.New()

	// Create a map to keep track of visited routes
	visited := make(map[string]bool)
	for k := range adjacentRoutes {
		visited[k] = false
	}

	// Add the starting route to the queue
	visited[srcRoute] = true
	queue.PushFront(srcRoute)

	// Breadth first search
	for queue.Len() != 0 {
		currentRoute := queue.Front().Value.(string)
		visited[currentRoute] = true
		queue.Remove(queue.Front())

		for _, childRoute := range adjacentRoutes[currentRoute] {
			if visited[childRoute] {
				continue
			}
			visited[childRoute] = true
			parents[childRoute] = currentRoute
			queue.PushBack(childRoute)

			if childRoute == destRoute {
				return parents
			}
		}
	}
	return nil
}

// Given potential starting and destination routes, find the set of routes that connect them
func getConnectingRoutes(fromRoutes []string, toRoutes []string, adjacentRoutes adjacencyMap) []string {
	var connections []string
	connectionFound := false
	for _, fromRoute := range fromRoutes {
		for _, toRoute := range toRoutes {

			// Don't bother with the breadth first search if the routes are the same
			if fromRoute == toRoute {
				connectionFound = true
				connections = append(connections, fromRoute)
				break
			}

			// Traverse the graph and reconstruct the shortest path between routes
			parents := buildTraversal(fromRoute, toRoute, adjacentRoutes)
			if len(parents) == 0 {
				continue
			}
			connectionFound = true
			curRoute := toRoute
			for curRoute != fromRoute {
				connections = append([]string{curRoute}, connections...)
				curRoute = parents[curRoute]
			}
			connections = append([]string{fromRoute}, connections...)
			break
		}

		// If a connection has been found, exit the loop, otherwise start over
		if connectionFound {
			break
		} else {
			connections = nil
		}
	}
	return connections
}

// For every route in the API response, find the route with the most or fewest stops
// Returns the route and its count. If there is a tie, it returns the first one found
func getRouteByStopCount(mbtaURL url.URL, routes routesResponse, fewest bool) (string, int, error) {
	stopCounts, err := getStopCounts(mbtaURL, routes)
	if err != nil {
		return "", 0, nil
	}

	route := ""
	stopCount := 0
	if fewest {
		stopCount = math.MaxInt32
	}
	for k, v := range stopCounts {
		if (fewest && v < stopCount) || (!fewest && v > stopCount) {
			stopCount = v
			route = k
		}
	}
	return route, stopCount, nil
}

// For every route in the API response, count all the stops for the route
// Returns a map from route long name to stop counts.
func getStopCounts(mbtaURL url.URL, routes routesResponse) (map[string]int, error) {
	stopCounts := make(map[string]int)
	for _, route := range routes.Data {
		routeStops, err := queryRouteStops(mbtaURL, route.ID)
		if err != nil {
			return nil, err
		}
		routeName := route.Attributes.LongName
		stopCounts[routeName] = len(routeStops.Data)
	}
	return stopCounts, nil
}

// For every route in the API response, create a mapping of stops to the routes they are on.
// If buildRouteAdjacencyMap is true, also construct the route adjacency list while mapping the stops
func getStopRoutes(mbtaURL url.URL, routes routesResponse, buildRouteAdjacencyMap bool) (stopMap, adjacencyMap, error) {
	stops := make(stopMap)
	adjacentRoutes := make(adjacencyMap)
	for _, route := range routes.Data {
		routeStops, err := queryRouteStops(mbtaURL, route.ID)
		if err != nil {
			return nil, nil, err
		}
		routeName := route.Attributes.LongName

		for _, stop := range routeStops.Data {
			// Traverse each stop and create an entry in the map to store the route its on
			stopName := stop.Attributes.Name
			if stops[stopName] == nil {
				stops[stopName] = new(stopData)
				stops[stopName].name = stop.Attributes.Name
			}
			stops[stopName].routes = append(stops[stopName].routes, routeName)

			if !buildRouteAdjacencyMap {
				continue
			}
			// If a stop has multiple routes, adjust the route adjacency list
			for _, curRoute := range stops[stopName].routes {
				if curRoute == routeName {
					continue
				}
				if !contains(adjacentRoutes[curRoute], routeName) {
					adjacentRoutes[curRoute] = append(adjacentRoutes[curRoute], routeName)
				}
				if !contains(adjacentRoutes[routeName], curRoute) {
					adjacentRoutes[routeName] = append(adjacentRoutes[routeName], curRoute)
				}
			}
		}
	}
	return stops, adjacentRoutes, nil
}

func main() {
	routesFlag := flag.Bool("routes", false, "Display the names of all the MBTA's subway lines.")
	longestRouteFlag := flag.Bool("longestRoute", false, "Display the subway route with the most stops and a count of its stops.")
	shortestRouteFlag := flag.Bool("shortestRoute", false, "Display the subway route with the fewest stops and a count of its stops.")
	transfersFlag := flag.Bool("transfers", false, "Display a list of all stops that are on multiple routes.")
	fromFlag := flag.String("from", "", "Your starting stop. Must be uesd with the --to flag.")
	toFlag := flag.String("to", "", "Your final stop. Must be paired with the --from flag.")
	listStopsFlag := flag.Bool("stops", false, "Lists all stops")
	flag.Parse()

	mbtaURL := url.URL{
		Scheme:   "https",
		Host:     "api-v3.mbta.com",
		RawQuery: "api_key=a65ddb1213ca4860b0495b76347c1ec1",
	}

	// Queries all the routes and their corresponding stops and prints this list of stops
	if *listStopsFlag {
		routes, err := querySubwayRoutes(mbtaURL)
		if err != nil {
			log.Fatal(err)
		}
		for _, route := range routes.Data {
			routeStops, err := queryRouteStops(mbtaURL, route.ID)
			if err != nil {
				log.Fatal(err)
			}
			for _, stop := range routeStops.Data {
				log.Println(stop.Attributes.Name)
			}
		}
	}

	// Prints the names of all the routes
	if *routesFlag {
		routes, err := querySubwayRoutes(mbtaURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Available subway routes: %s", strings.Join(getLongNames(&routes), ", "))
	}

	// Helper function for finding the shortest or longest route
	getRoute := func(fewest bool) {
		routes, err := querySubwayRoutes(mbtaURL)
		if err != nil {
			log.Fatal(err)
		}

		route, count, err := getRouteByStopCount(mbtaURL, routes, fewest)
		if err != nil {
			log.Fatal(err)
		}

		routeQuantifier := "longest"
		if fewest {
			routeQuantifier = "shortest"
		}

		if route == "" {
			log.Printf("No %s subway route found", routeQuantifier)
		} else {
			log.Printf("The %s route is: %s with %d stops", routeQuantifier, route, count)
		}
	}

	// Finds the routes with the most stops
	if *longestRouteFlag {
		getRoute(false)
	}

	// Finds the routes with the fewest stops
	if *shortestRouteFlag {
		getRoute(true)
	}

	// Finds the routes with the most stops
	if *transfersFlag {
		routes, err := querySubwayRoutes(mbtaURL)
		if err != nil {
			log.Fatal(err)
		}

		stops, _, err := getStopRoutes(mbtaURL, routes, false)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Stops with transfers: ")
		for _, v := range stops {
			if len(v.routes) > 1 {
				log.Printf("   %s: %v", v.name, v.routes)
			}
		}
	}

	// Finds the connecting routes between 2 stops
	from := *fromFlag
	to := *toFlag
	if (from != "" && to == "") || (from == "" && to != "") {
		log.Println("The --to and --from commands must be used together and have values")
	} else if from != "" && to != "" {
		routes, err := querySubwayRoutes(mbtaURL)
		if err != nil {
			log.Fatal(err)
		}

		// Get the stops and create the adjacency list
		stops, adjacentRoutes, err := getStopRoutes(mbtaURL, routes, true)
		if err != nil {
			log.Fatal(err)
		}

		if stops[from] == nil {
			log.Fatalf("--from stop %s does not exist", from)
		}
		if stops[to] == nil {
			log.Fatalf("--to stop %s does not exist", to)
		}

		// Find the path between routes and print the results
		fromRoutes := stops[from].routes
		toRoutes := stops[to].routes
		connections := getConnectingRoutes(fromRoutes, toRoutes, adjacentRoutes)
		if len(connections) == 0 {
			log.Fatal("No connecting route!")
		}
		log.Printf("%s to %s: %v", from, to, connections)
	}
}
