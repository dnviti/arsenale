package cmd

var gatewayColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "STATUS", Field: "operationalStatus"},
	{Header: "HEALTHY", Field: "healthyInstances"},
	{Header: "RUNNING", Field: "runningInstances"},
	{Header: "DESIRED", Field: "desiredReplicas"},
}

var gatewayInstanceColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "STATUS", Field: "status"},
	{Header: "HEALTH", Field: "healthStatus"},
	{Header: "HOST", Field: "host"},
	{Header: "PORT", Field: "port"},
	{Header: "CONTAINER", Field: "containerName"},
}

var gatewayStatusColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "STATUS", Field: "operationalStatus"},
	{Header: "HEALTHY", Field: "healthyInstances"},
	{Header: "RUNNING", Field: "runningInstances"},
	{Header: "DESIRED", Field: "desiredReplicas"},
	{Header: "TUNNEL", Field: "tunnelConnected"},
	{Header: "DETAIL", Field: "operationalReason"},
}

var gatewayTemplateColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
}
