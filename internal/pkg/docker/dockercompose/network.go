package dockercompose

import (
	"bytes"
	_ "embed"
	"fmt"
	compose "github.com/compose-spec/compose-go/types"
	"strings"
	"text/template"
)

//go:embed templates/service-discovery-record.yml
var serviceDiscoveryTemplate string

type aliasLink struct {
	AliasName string
	TargetSvc string
}

// serviceLinkageAddon produces a CloudFormation addon producing Route53 CNAME aliases to service discovery endpoints.
func serviceLinkageAddon(service *compose.ServiceConfig, otherSvcs compose.Services) (string, error) {
	linked, err := findLinkedServices(service, otherSvcs)
	if err != nil {
		return "", err
	}

	aliasLinks := serviceDiscoveryAliases(linked)

	tmpl := template.New("service-discovery-record.yml")
	_, err = tmpl.Parse(serviceDiscoveryTemplate)
	if err != nil {
		return "", fmt.Errorf("parse service discovery addon template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, aliasLinks)
	if err != nil {
		return "", fmt.Errorf("evaluate service discovery addon template: %w", err)
	}

	return buf.String(), nil
}

func serviceDiscoveryAliases(linked map[string]compose.ServiceConfig) []aliasLink {
	var aliasLinks []aliasLink

	for alias, svc := range linked {
		aliasLinks = append(aliasLinks, aliasLink{
			AliasName: alias,
			TargetSvc: svc.Name,
		})
	}

	return aliasLinks
}

// findLinkedServices uses Compose networking rules to determine the other services that this service can talk to.
func findLinkedServices(service *compose.ServiceConfig, otherSvcs compose.Services) (map[string]compose.ServiceConfig, error) {
	switch {
	case service.NetworkMode == "none":
		return nil, nil
	case strings.HasPrefix(service.NetworkMode, "service:"):
		svcName := strings.TrimPrefix(service.NetworkMode, "service:")
		for _, svc := range otherSvcs {
			if svc.Name == svcName {
				otherSvcs = []compose.ServiceConfig{svc}
				break
			}
		}
	case service.NetworkMode == "host":
		return nil, fmt.Errorf("network mode \"%s\" is not supported", service.NetworkMode)
	}

	renames := make(map[string]string)
	for _, link := range service.Links {
		parts := strings.Split(link, ":")

		if len(parts) == 2 {
			renames[parts[0]] = parts[1]
		}
	}

	if len(service.Networks) != 0 {
		networks := make(map[string]bool)
		for name := range service.Networks {
			networks[name] = true
			// TODO: Check unsupported
		}

		var netSvcs []compose.ServiceConfig
		for _, svc := range otherSvcs {
			for net := range svc.Networks {
				if networks[net] {
					netSvcs = append(netSvcs, svc)
					break
				}
			}
		}

		otherSvcs = netSvcs
	}

	linked := make(map[string]compose.ServiceConfig)

	for _, svc := range otherSvcs {
		alias := renames[svc.Name]
		if alias == "" {
			alias = svc.Name
		}
		linked[alias] = svc
	}

	return linked, nil
}
