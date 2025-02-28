# Loadbalancer

Brief description of what the project does.

## Table of Contents

- [Loadbalancer](#loadbalancer)
	- [Table of Contents](#table-of-contents)
	- [Features](#features)

## Features

    - Configurable
    - Healthcheck
    - Strategy pattern
    	- round robin - even distribution, always getting next server in order
    	- weighted round robin - using server weights for indicating the distribution cycle. If loadbalancer is forwarding requests to 3 servers, where they each have 5-3-2 weights, one cycle would be finished after 10 requests. 5 requests would be forwarded to server A, 3 to B and 2 to C. We track current weights (initially using configured weights) and index of current active server. Every time new server gets selected, current weight will be reduced by one. When all servers have 0 current weight, then we reset them to their configured values.
    	- smooth weighted round robin - next server is the server with current amount of weights
    	- sticky session -
    	- least connections
    	- ip hash
    - Rate limit
