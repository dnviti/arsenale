package importexportapi

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type columnMapping map[string]string

type mremoteConnection struct {
	Name        string
	Hostname    string
	Port        string
	Protocol    string
	Username    string
	Password    string
	Description string
	Panel       string
}

type mremoteRoot struct {
	Nodes []mremoteNode `xml:",any"`
}

type mremoteNode struct {
	XMLName         xml.Name      `xml:""`
	NameAttr        string        `xml:"Name,attr"`
	HostnameAttr    string        `xml:"Hostname,attr"`
	HostAttr        string        `xml:"Host,attr"`
	PortAttr        string        `xml:"Port,attr"`
	ProtocolAttr    string        `xml:"Protocol,attr"`
	UsernameAttr    string        `xml:"Username,attr"`
	PasswordAttr    string        `xml:"Password,attr"`
	DescriptionAttr string        `xml:"Description,attr"`
	PanelAttr       string        `xml:"Panel,attr"`
	Name            string        `xml:"Name"`
	Hostname        string        `xml:"Hostname"`
	Host            string        `xml:"Host"`
	Port            string        `xml:"Port"`
	Protocol        string        `xml:"Protocol"`
	Username        string        `xml:"Username"`
	Password        string        `xml:"Password"`
	Description     string        `xml:"Description"`
	Panel           string        `xml:"Panel"`
	Children        []mremoteNode `xml:",any"`
}

type rdpConnection struct {
	FullAddress string
	Hostname    string
	Port        int
	Username    string
}

func parseColumnMapping(raw string) (columnMapping, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, &requestError{status: 400, message: "columnMapping must be valid JSON"}
	}

	result := make(columnMapping, len(payload))
	for key, value := range payload {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		normalizedValue := strings.ToLower(strings.TrimSpace(value))
		if normalizedKey != "" && normalizedValue != "" {
			result[normalizedKey] = normalizedValue
		}
	}
	return result, nil
}

func (m columnMapping) resolve(key, fallback string) string {
	if m != nil {
		if value := strings.TrimSpace(m[strings.ToLower(strings.TrimSpace(key))]); value != "" {
			return value
		}
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func parseRDPFile(content string) rdpConnection {
	properties := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 3)
		if len(parts) != 3 {
			continue
		}
		properties[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[2])
	}

	fullAddress := firstNonEmpty(properties["full address"], properties["address"])
	hostname, port := parseRDPAddress(fullAddress)
	return rdpConnection{
		FullAddress: fullAddress,
		Hostname:    hostname,
		Port:        port,
		Username:    properties["username"],
	}
}

func parseRDPAddress(value string) (string, int) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", 3389
	}
	idx := strings.LastIndex(trimmed, ":")
	if idx >= 0 {
		port, err := strconv.Atoi(strings.TrimSpace(trimmed[idx+1:]))
		if err == nil && port >= 1 && port <= 65535 {
			return strings.TrimSpace(trimmed[:idx]), port
		}
	}
	return trimmed, 3389
}

func parseMRemoteNGXML(raw string) ([]mremoteConnection, error) {
	var root mremoteRoot
	if err := xml.NewDecoder(bytes.NewBufferString(raw)).Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to parse mRemoteNG XML: %w", err)
	}

	connections := make([]mremoteConnection, 0)
	for _, node := range root.Nodes {
		walkMRemoteNode(node, &connections)
	}
	return connections, nil
}

func walkMRemoteNode(node mremoteNode, dest *[]mremoteConnection) {
	protocol := firstNonEmpty(node.ProtocolAttr, node.Protocol)
	if protocol != "" {
		*dest = append(*dest, mremoteConnection{
			Name:        firstNonEmpty(node.NameAttr, node.Name, "Unnamed"),
			Hostname:    firstNonEmpty(node.HostnameAttr, node.HostAttr, node.Hostname, node.Host),
			Port:        firstNonEmpty(node.PortAttr, node.Port, defaultMRemotePort(protocol)),
			Protocol:    protocol,
			Username:    firstNonEmpty(node.UsernameAttr, node.Username),
			Password:    firstNonEmpty(node.PasswordAttr, node.Password),
			Description: firstNonEmpty(node.DescriptionAttr, node.Description),
			Panel:       firstNonEmpty(node.PanelAttr, node.Panel),
		})
	}
	for _, child := range node.Children {
		walkMRemoteNode(child, dest)
	}
}

func defaultMRemotePort(protocol string) string {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "RDP":
		return "3389"
	case "SSH", "TELNET", "SFTP", "SCP":
		return "22"
	case "VNC":
		return "5900"
	case "HTTP":
		return "80"
	case "HTTPS":
		return "443"
	default:
		return "22"
	}
}

func mapMRemoteProtocol(protocol string) string {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "RDP", "RDP2":
		return "RDP"
	case "SSH", "TELNET", "SFTP", "SCP", "RAWS":
		return "SSH"
	case "VNC":
		return "VNC"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
