package clinicapi

import "testing"

func TestParsePortalURLExtractsBaseURLAndClusterIDFromUserURLs(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		baseURL   string
		clusterID string
	}{
		{
			name:      "cloud portal",
			raw:       "https://clinic.pingcap.com/portal/#/orgs/1372813089196930499/clusters/10989049060142230334",
			baseURL:   "https://clinic.pingcap.com",
			clusterID: "10989049060142230334",
		},
		{
			name:      "cn tiup portal",
			raw:       "https://clinic.pingcap.com.cn/portal/#/orgs/1075/clusters/7460723698814898616?from=1767679200&to=1767682800",
			baseURL:   "https://clinic.pingcap.com.cn",
			clusterID: "7460723698814898616",
		},
		{
			name:      "global tiup portal",
			raw:       "https://clinic.pingcap.com/portal/#/orgs/1372813089196930348/clusters/7372714695339837431?from=1773547800&to=1773548100",
			baseURL:   "https://clinic.pingcap.com",
			clusterID: "7372714695339837431",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParsePortalURL(tt.raw)
			if err != nil {
				t.Fatalf("ParsePortalURL failed: %v", err)
			}
			if info.BaseURL != tt.baseURL {
				t.Fatalf("unexpected base URL: %s", info.BaseURL)
			}
			if info.ClusterID != tt.clusterID {
				t.Fatalf("unexpected cluster id: %s", info.ClusterID)
			}
		})
	}
}

func TestParsePortalURLRejectsMissingClusterRoute(t *testing.T) {
	_, err := ParsePortalURL("https://clinic.pingcap.com/portal/#/orgs/org-1")
	if err == nil {
		t.Fatalf("expected missing cluster route to fail")
	}
	if ClassOf(err) != ErrInvalidRequest {
		t.Fatalf("expected invalid request error, got %v", err)
	}
}
