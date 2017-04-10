package cloudfoundry

import (
	"encoding/json"
)

// ServicePlan object.
type ServicePlan struct {
	Name        string       `json:"name"`
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Metadata    PlanMetadata `json:"metadata, omitempty"`
	Free        bool         `json:"free, omitempty"`
}

// Service object.
type Service struct {
	Name            string          `json:"name"`
	ID              string          `json:"id"`
	Description     string          `json:"description"`
	Bindable        bool            `json:"bindable"`
	PlanUpdateable  bool            `json:"plan_updateable, omitempty"`
	Tags            []string        `json:"tags, omitempty"`
	Requires        []string        `json:"requires, omitempty"`
	Metadata        ServiceMetadata `json:"metadata, omitempty"`
	Plans           []ServicePlan   `json:"plans"`
	DashboardClient interface{}     `json:"dashboard_client"`
}

// Catalog object.
type Catalog struct {
	Services []Service `json:"services"`
}

// ServiceBinding object.
type ServiceBinding struct {
	ID                string `json:"id"`
	ServiceID         string `json:"service_id"`
	AppID             string `json:"app_id"`
	ServicePlanID     string `json:"service_plan_id"`
	PrivateKey        string `json:"private_key"`
	ServiceInstanceID string `json:"service_instance_id"`
}

// CreateServiceBindingResponse object.
type CreateServiceBindingResponse struct {
	Credentials interface{} `json:"credentials"`
}

// Provider object.
type Provider struct {
	Name string `json:"name"`
}

// Listing object.
type Listing struct {
	ImageURL    string `json:"imageUrl"`
	Blurb       string `json:"blurb"`
	Description string `json:"longDescription"`
}

// ServiceMetadata object.
type ServiceMetadata struct {
	Provider         Provider `json:"provider"`
	Listing          Listing  `json:"listing"`
	DisplayName      string   `json:"displayName"`
	DocumentationURL string   `json:"documentationUrl,omitempty"`
}

// PlanMetadata object.
type PlanMetadata struct {
	Cost    float64  `json:"cost"`
	Bullets []string `json:"bullets"`
}

// Credential object.
type Credential struct {
	PublicIP   string `json:"public_ip"`
	Username   string `json:"username"`
	PrivateKey string `json:"private_key"`
}

// ServiceInstance object.
type ServiceInstance struct {
	ID               string         `json:"id"`
	DashboardURL     string         `json:"dashboard_url"`
	InternalID       string         `json:"internalId, omitempty"`
	ServiceID        string         `json:"service_id"`
	PlanID           string         `json:"plan_id"`
	OrganizationGUID string         `json:"organization_guid"`
	SpaceGUID        string         `json:"space_guid"`
	LastOperation    *LastOperation `json:"last_operation, omitempty"`
	Parameters       interface{}    `json:"parameters, omitempty"`
}

// LastOperation object.
type LastOperation struct {
	State                    string `json:"state"`
	Description              string `json:"description"`
	AsyncPollIntervalSeconds int    `json:"async_poll_interval_seconds, omitempty"`
}

// CreateServiceInstanceResponse object.
type CreateServiceInstanceResponse struct {
	DashboardURL  string         `json:"dashboard_url"`
	LastOperation *LastOperation `json:"last_operation, omitempty"`
}

// GetCatalog populates and returns a new Catalog object.
func GetCatalog() Catalog {
	c := Catalog{
		Services: []Service{
			{
				ID:          "mydis-18f31cee-ef2a-4351-92f4-926846f8e736",
				Name:        "Mydis",
				Description: "Distributed, reliable database and cache library, server, and client. Inspired by Redis.",
				Requires:    []string{"volume_mount"},
				Tags:        []string{"cache", "database", "nosql", "key-value"},
				Bindable:    true,
				Metadata: ServiceMetadata{
					Provider: Provider{Name: "Ross Peoples"},
					Listing: Listing{
						ImageURL:    "https://github.com/deejross/mydis/raw/master/logo/Mydis.png",
						Blurb:       "Distributed, reliable database and cache library, server, and client. Inspired by Redis.",
						Description: "Distributed, reliable database and cache library, server, and client. Inspired by Redis, Mydis is written entirely in Go and can be used as a library, embedded into an existing application, or as a standalone client/server.",
					},
					DisplayName:      "Mydis Service",
					DocumentationURL: "https://github.com/deejross/mydis",
				},
				PlanUpdateable: true,
				Plans: []ServicePlan{
					{
						ID:          "mydis-dev-18f31cee-ef2a-4351-92f4-926846f8e736",
						Name:        "dev",
						Description: "Single, standalone instance of Mydis",
						Free:        true,
						Metadata: PlanMetadata{
							Bullets: []string{
								"Single instance",
								"Ideal for development environments",
							},
						},
					},
					{
						ID:          "mydis-prod-18f31cee-ef2a-4351-92f4-926846f8e736",
						Name:        "prod",
						Description: "Three node, highly available cluster for Mydis",
						Free:        true,
						Metadata: PlanMetadata{
							Bullets: []string{
								"Three node cluster",
								"High availability",
								"Ideal for production environments",
							},
						},
					},
				},
			},
		},
	}

	return c
}

// JSON marshals the Catalog into a JSON byte slice.
func (c Catalog) JSON() []byte {
	b, _ := json.Marshal(&c.Services)
	return b
}
