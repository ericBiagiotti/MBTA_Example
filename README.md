# MBTA_Example

Written by Eric Biagiotti 12/31/2019

## Build and Run Instructions ##

* Built and tested with go 1.13.5.
* Install go by following the instructions here: https://golang.org/doc/install
* By default, go looks for code in a specific directory. It is configurable by setting the `GOPATH` environment variable, but the easiest way to get thing working is to create the following folder on Windows `%USERPROFILE%/go/src/`, or `~/go/src/` in Linux. Then you can run `git clone https://github.com/ericBiagiotti/MBTA_Example.git` from that folder.
* From the `MBTA_Example` folder, open a terminal and run `go run MBTA_Example` or `go build` and run the executable directly.
* Running `go test` from the MBTA_Example folder will run tests.
* Use the `-h` flag to see other potential flags. For example: `go run MBTA_Example --routes --longestRoute --shortestRoute --transfers` will answer questions 1 and 2.
* Use the `--from` and `--to` flags to experiment with question 3. These flags expect the full name of the stop. If there is a space in the name, use quotes. Use the `--stops` flag to see a list of potential stops.

## Design ##

### Queries ###
 I use two queries to answer the questions: 1) get all the routes filtered by type and 2) get all the stops filtered by route. I chose to use filters in the queries to reduce the size of the response and client parsing logic.

 I separated some of the response data into several structs to make testing easier and separated the GET requests and JSON marshalling into a generic function. Any errors returned by the GET requests or the JSON marshalling is returned to the caller.

### Question 1 ###
I query the API for all routes, filtered by type, and use a helper function `getLongNames` to loop over the response and create a list of long names.

### Question 2: Parts 1 and 2 ###
I couldn't get stop queries for multiple routes to work. For example:

`https://api-v3.mbta.com/stops?include=route&filter[route]=Red` - Populates the `route` attribute in the response
`https://api-v3.mbta.com/stops?include=route&filter[route]=Red,Blue` - I recieve `"route":{"data":null}`.

Subsequently, I had to query stops for each individual route in the `getStopCounts` function. `getRouteByStopCount`, uses `getStopCounts` and creates a hash from route long name to stop counts. The time complexity for this operation is `O(R)` where `R` is the number of routes. We loop through the routes twice, once to get the total number of stop on a route, and once to find the route with the most or fewest stops.

### Question 2: Part 3 ###
I followed a dead end for a bit on this one. I saw `recommended_trasfers` in the stops endpoint responses and tried to get that to work, but that is something completely different. Instead, I wrote `getStopRoutes`, which creates a hash of stops to the routes they are on, which is populated as we loop through all the stops for each route. The time complexity for this operation is `O(S + T)` where `S` is the total number of stops, and `T` is the number of transfers. In other words, we have to look at every stop and if a stop is on multiple routes, we will look at it multiple times.

### Question 3 ###
Because we only need to know the connecting routes between stops, this can be thought of as an undirected graph problem, where each route is a node and if a route shares a stop, there is an edge between them.

The solution uses an adjacency list, which is a hash representations of graphs. For our example, the hash would be keyed by the route name and the value is all the routes it is adjacent to (other routes that share a stop). This is convenient because we already create a hash of stops to routes in `getStopRoutes`, so I modified it to also build the adjacency list.

Given 2 stops from the user, `getConnectingRoutes` iterates over the combinations of starting and destination routes and `buildTraversal` does a breadth-first search of the graph and keeps track of visited routes. `buildTraversal` was translated from https://www.geeksforgeeks.org/shortest-path-unweighted-graph/.

The time complexity for the operation is `O(S * D * (V + E))` where
* `S` is the number of starting routes
* `D` is the number of destination routes
* `V` is the number of routes, or vertices on the graph
* `E ` is the number of directly connected routes, or edges on the graph

## Testing ##
Each function used to answer the 3 questions is unit tested. The test suite creates a test server with mocked responses for functions that query the MBTA API. Another potential approach would be to pass function pointers where `querySubwayRoutes` and `queryRouteStops` are used. This would improve unit test isolation and avoid overhead associated with running the test server and parsing the JSON where JSON parsing is not being explicitly tested.

## Problems and Potential Improvements ##
- Adjacency list generation in `getStopRoutes` should probably be in a seperate function for clarity.
- I am currently using the names of routes and stops as hash keys to simplify things, but this should really be using IDs.
- Add more edge case and error testing. For example, `getConnectingRoutes` iterates over the combinations of starting and destination routes for any given pair of stops, but the first set of routes will always find a connection given the current MBTA graph. Consequently, the code for handling the case when the first pair of routes don't have a connection but subsequent pairs do is not fully tested.
- The `http.Get()` function doesn't return an error for non 2xx response codes, so `getAndDecode` should check for different response codes from the MBTA API and log unexpected results.
- Client side caching of data. The MBTA resources we are concerned with for this example are fairly static. They provide a way to check API v3 supports caching via the `Last-Modified` response and  `If-Modified-Since` request headers.
