package models

import (
	"encoding/json"
	"sync"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Name      string `json:"name" yaml:"name"`
	URL       string `json:"url" yaml:"url"`
	HealthURL string `json:"health_url" yaml:"health_url"`
	Healthy   bool   `json:"healthy"`
	Weight    int    `json:"weight"`
	sync.RWMutex
}

func (s *Server) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		HealthURL string `json:"health_url"`
		Healthy   *bool  `json:"healthy"`
		Weight    int    `json:"weight"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.Name = aux.Name
	s.URL = aux.URL
	s.HealthURL = aux.HealthURL
	s.Weight = aux.Weight

	if aux.Healthy == nil {
		s.Healthy = true
	} else {
		s.Healthy = *aux.Healthy
	}

	return nil
}

func (s *Server) UnmarshalYAML(data []byte) error {
	aux := &struct {
		Name      string `yaml:"name"`
		URL       string `yaml:"url"`
		HealthURL string `yaml:"health_url"`
		Healthy   *bool  `yaml:"Healthy"`
		Weight    int    `yaml:"weight"`
	}{}

	if err := yaml.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.Name = aux.Name
	s.URL = aux.URL
	s.HealthURL = aux.HealthURL
	s.Weight = aux.Weight

	if aux.Healthy == nil {
		s.Healthy = true
	} else {
		s.Healthy = *aux.Healthy
	}

	return nil
}
