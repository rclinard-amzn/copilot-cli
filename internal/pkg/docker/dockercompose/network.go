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
	allSvcs := make(compose.Services, 0, 1+len(otherSvcs))
	allSvcs = append(allSvcs, *service)
	allSvcs = append(allSvcs, otherSvcs...)

	switch {
	case service.NetworkMode == "none":
		return nil, nil
	case strings.HasPrefix(service.NetworkMode, "service:"):
		svcName := strings.TrimPrefix(service.NetworkMode, "service:")
		var linked *compose.ServiceConfig
		for _, svc := range allSvcs {
			if svc.Name == svcName {
				linked = &svc
				break
			}
		}
		if linked == nil {
			return nil, fmt.Errorf("no service with the name \"%s\" found for network mode \"%s\"", svcName, service.NetworkMode)
		}
		allSvcs = []compose.ServiceConfig{*linked}
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

	networks := make(map[string]bool)

	if len(service.Networks) == 0 {
		networks["default"] = true
	} else {
		for name := range service.Networks {
			networks[name] = true
			// TODO: Check unsupported
		}
	}

	var netSvcs []compose.ServiceConfig
	for _, svc := range allSvcs {
		for net := range svc.Networks {
			if networks[net] {
				netSvcs = append(netSvcs, svc)
				break
			}
		}

		if len(svc.Networks) == 0 && networks["default"] {
			netSvcs = append(netSvcs, svc)
		}
	}

	allSvcs = netSvcs

	linked := make(map[string]compose.ServiceConfig)

	for _, svc := range allSvcs {
		alias := renames[svc.Name]
		if alias == "" {
			alias = svc.Name
		}
		linked[alias] = svc
	}

	return linked, nil
}
