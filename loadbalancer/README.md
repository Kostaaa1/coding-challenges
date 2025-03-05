# Loadbalancer

Brief description of what the project does.

## Table of Contents

- [Loadbalancer](#loadbalancer)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)

## Features

- Config watcher - watching for changes in main config file (JSON | YAML), then it sends the SIGHUP signal which will update the config, removing the need of resetting the lb.
- Healthchecker -
- Strategies - (round_robin, weighted_round_robin, smooth_weighted_round_robin, random (supports weights), sticky_session)
- CBR - Content based routing
- [TODO] - Rate limit
- [TODO] - Add TLS
- [TODO] - Add response caching
- Docker

## Strategies

- Round robin -
- Weighted round robin -
- Smooth weighted round robin -
- Random -
- Sticky sessions -
- Least connection
