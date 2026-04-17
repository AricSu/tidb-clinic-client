package clinicapi

import "github.com/AricSu/tidb-clinic-client/internal/clinic"

type Client = clinic.Client

func NewClient(baseURL string, opts ...ClientOpt) (*Client, error) {
	return clinic.NewClient(baseURL, opts...)
}
func NewClientWithConfig(cfg Config) (*Client, error) {
	return clinic.NewClientWithConfig(cfg)
}
