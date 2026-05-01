package syncprofiles

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

func testNetBoxConnection(config syncProfileConfig, apiToken string) (bool, string) {
	u, err := neturl.Parse(config.URL)
	if err != nil {
		return false, err.Error()
	}
	statusURL := u.ResolveReference(&neturl.URL{Path: "/api/status/"}).String()
	req, err := http.NewRequest(http.MethodGet, statusURL, nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Token "+apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, ""
	}
	return false, fmt.Sprintf("NetBox returned HTTP %d", resp.StatusCode)
}

func discoverNetBoxDevices(config syncProfileConfig, apiToken string) ([]discoveredDevice, error) {
	devices := make([]discoveredDevice, 0)

	physicalDevices, err := fetchAllPages[netBoxDevice](config.URL, "/api/dcim/devices/", apiToken, config.Filters)
	if err != nil {
		return nil, err
	}
	for _, dev := range physicalDevices {
		ip := resolveIP(dev.PrimaryIP4, dev.PrimaryIP6)
		if ip == "" {
			continue
		}
		protocol := resolveProtocol(dev.Platform, config)
		port := resolvePort(protocol, config.DefaultPort)
		rackName := ""
		if dev.Rack != nil {
			rackName = dev.Rack.Name
		} else if dev.Location != nil {
			rackName = dev.Location.Name
		}
		siteName := ""
		if dev.Site != nil {
			siteName = dev.Site.Name
		}
		devices = append(devices, discoveredDevice{
			ExternalID:  fmt.Sprintf("device:%d", dev.ID),
			Name:        defaultDisplayName(dev.Name, dev.Display),
			Host:        ip,
			Port:        port,
			Protocol:    protocol,
			SiteName:    siteName,
			RackName:    rackName,
			Description: strings.TrimSpace(dev.Description),
			Metadata: map[string]any{
				"type":         "device",
				"netboxId":     dev.ID,
				"platform":     platformSlug(dev.Platform),
				"status":       statusValue(dev.Status),
				"customFields": dev.CustomFields,
			},
		})
	}

	vms, err := fetchAllPages[netBoxVM](config.URL, "/api/virtualization/virtual-machines/", apiToken, config.Filters)
	if err != nil {
		return nil, err
	}
	for _, vm := range vms {
		ip := resolveIP(vm.PrimaryIP4, vm.PrimaryIP6)
		if ip == "" {
			continue
		}
		protocol := resolveProtocol(vm.Platform, config)
		port := resolvePort(protocol, config.DefaultPort)
		siteName := ""
		if vm.Site != nil {
			siteName = vm.Site.Name
		}
		rackName := ""
		if vm.Cluster != nil {
			rackName = vm.Cluster.Name
		}
		devices = append(devices, discoveredDevice{
			ExternalID:  fmt.Sprintf("vm:%d", vm.ID),
			Name:        defaultDisplayName(vm.Name, vm.Display),
			Host:        ip,
			Port:        port,
			Protocol:    protocol,
			SiteName:    siteName,
			RackName:    rackName,
			Description: strings.TrimSpace(vm.Description),
			Metadata: map[string]any{
				"type":         "vm",
				"netboxId":     vm.ID,
				"platform":     platformSlug(vm.Platform),
				"status":       statusValue(vm.Status),
				"customFields": vm.CustomFields,
			},
		})
	}

	return devices, nil
}

func fetchAllPages[T any](baseURL, path, apiToken string, filters map[string]string) ([]T, error) {
	values := neturl.Values{}
	values.Set("limit", "100")
	for key, value := range filters {
		values.Set(key, value)
	}
	startURL, err := neturl.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	current := startURL.ResolveReference(&neturl.URL{Path: path, RawQuery: values.Encode()}).String()
	client := &http.Client{Timeout: 30 * time.Second}

	result := make([]T, 0)
	for current != "" {
		req, err := http.NewRequest(http.MethodGet, current, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Token "+apiToken)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var page netBoxPaginatedResponse[T]
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				io.Copy(io.Discard, resp.Body)
				err = fmt.Errorf("NetBox returned HTTP %d", resp.StatusCode)
				return
			}
			err = json.NewDecoder(resp.Body).Decode(&page)
		}()
		if err != nil {
			return nil, err
		}
		result = append(result, page.Results...)
		if page.Next != nil {
			current = *page.Next
		} else {
			current = ""
		}
	}
	return result, nil
}

func resolveIP(ip4, ip6 *netBoxIP) string {
	if ip4 != nil && strings.TrimSpace(ip4.Address) != "" {
		return stripCIDR(ip4.Address)
	}
	if ip6 != nil && strings.TrimSpace(ip6.Address) != "" {
		return stripCIDR(ip6.Address)
	}
	return ""
}

func stripCIDR(address string) string {
	if idx := strings.Index(address, "/"); idx >= 0 {
		return address[:idx]
	}
	return address
}

func resolveProtocol(platform *netBoxPlatform, config syncProfileConfig) string {
	if platform != nil {
		if mapped, ok := config.PlatformMapping[platform.Slug]; ok && strings.TrimSpace(mapped) != "" {
			return mapped
		}
	}
	return config.DefaultProtocol
}

func resolvePort(protocol string, defaults map[string]int) int {
	if defaults != nil {
		if value, ok := defaults[protocol]; ok && value > 0 {
			return value
		}
	}
	switch protocol {
	case "RDP":
		return 3389
	case "VNC":
		return 5900
	default:
		return 22
	}
}

func defaultDisplayName(name, display string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return display
}

func platformSlug(platform *netBoxPlatform) string {
	if platform == nil {
		return ""
	}
	return platform.Slug
}

func statusValue(status *netBoxStatus) string {
	if status == nil {
		return ""
	}
	return status.Value
}
