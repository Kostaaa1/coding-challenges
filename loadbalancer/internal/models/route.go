package models

type Routing struct {
	Enabled       bool        `json:"enabled" yaml:"enabled"`
	DefaultServer string      `json:"default_server" yaml:"default_server"`
	Rules         []RouteRule `json:"rules" yaml:"rules"`
}

type RouteRule struct {
	Conditions []RouteCondition `json:"conditions" yaml:"conditions"`
	Action     RouteAction      `json:"action" yaml:"action"`
}

type RouteCondition struct {
	PathPrefix string            `json:"path_prefix,omitempty" yaml:"path_prefix,omitempty"`
	Method     string            `json:"method,omitempty" yaml:"method,omitempty"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

type RouteAction struct {
	RouteTo string `json:"route_to" yaml:"route_to"`
}
