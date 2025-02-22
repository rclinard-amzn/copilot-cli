//go:build integration || localintegration

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package stack_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/copilot-cli/internal/pkg/manifest"

	"gopkg.in/yaml.v3"

	"github.com/aws/copilot-cli/internal/pkg/deploy"
	"github.com/aws/copilot-cli/internal/pkg/deploy/cloudformation/stack"
	"github.com/stretchr/testify/require"
)

func TestEnvStack_Template(t *testing.T) {
	testCases := map[string]struct {
		input          *deploy.CreateEnvironmentInput
		wantedFileName string
	}{
		"generate template with embedded manifest file with container insights and imported certificates": {
			input: func() *deploy.CreateEnvironmentInput {
				rawMft := `name: test
type: Environment
# Create the public ALB with certificates attached.
http:
  public:
    certificates:
      - cert-1
      - cert-2
observability:
  container_insights: true # Enable container insights.`
				var mft manifest.Environment
				err := yaml.Unmarshal([]byte(rawMft), &mft)
				require.NoError(t, err)
				return &deploy.CreateEnvironmentInput{
					Version: "1.x",
					App: deploy.AppInformation{
						AccountPrincipalARN: "arn:aws:iam::000000000:root",
						Name:                "demo",
					},
					Name:                 "test",
					ArtifactBucketARN:    "arn:aws:s3:::mockbucket",
					ArtifactBucketKeyARN: "arn:aws:kms:us-west-2:000000000:key/1234abcd-12ab-34cd-56ef-1234567890ab",
					CustomResourcesURLs: map[string]string{
						"CertificateValidationFunction": "https://mockbucket.s3-us-west-2.amazonaws.com/dns-cert-validator",
						"DNSDelegationFunction":         "https://mockbucket.s3-us-west-2.amazonaws.com/dns-delegation",
						"CustomDomainFunction":          "https://mockbucket.s3-us-west-2.amazonaws.com/custom-domain",
					},
					AllowVPCIngress: true,
					Mft:             &mft,
					RawMft:          []byte(rawMft),
				}
			}(),
			wantedFileName: "template-with-imported-certs-observability.yml",
		},
		"generate template with custom resources": {
			input: func() *deploy.CreateEnvironmentInput {
				rawMft := `name: test
type: Environment`
				var mft manifest.Environment
				err := yaml.Unmarshal([]byte(rawMft), &mft)
				require.NoError(t, err)
				return &deploy.CreateEnvironmentInput{
					Version: "1.x",
					App: deploy.AppInformation{
						AccountPrincipalARN: "arn:aws:iam::000000000:root",
						Name:                "demo",
					},
					Name:                 "test",
					ArtifactBucketARN:    "arn:aws:s3:::mockbucket",
					ArtifactBucketKeyARN: "arn:aws:kms:us-west-2:000000000:key/1234abcd-12ab-34cd-56ef-1234567890ab",
					CustomResourcesURLs: map[string]string{
						"CertificateValidationFunction": "https://mockbucket.s3-us-west-2.amazonaws.com/dns-cert-validator",
						"DNSDelegationFunction":         "https://mockbucket.s3-us-west-2.amazonaws.com/dns-delegation",
						"CustomDomainFunction":          "https://mockbucket.s3-us-west-2.amazonaws.com/custom-domain",
					},
					AllowVPCIngress: true,
					Mft:             &mft,
					RawMft:          []byte(rawMft),
				}
			}(),
			wantedFileName: "template-with-basic-manifest.yml",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// GIVEN
			wanted, err := os.ReadFile(filepath.Join("testdata", "environments", tc.wantedFileName))
			require.NoError(t, err, "read wanted template")
			wantedObj := make(map[any]any)
			require.NoError(t, yaml.Unmarshal(wanted, wantedObj))

			// WHEN
			envStack := stack.NewEnvStackConfig(tc.input)
			actual, err := envStack.Template()
			require.NoError(t, err, "serialize template")
			actualObj := make(map[any]any)
			require.NoError(t, yaml.Unmarshal([]byte(actual), actualObj))
			actualMetadata := actualObj["Metadata"].(map[string]any) // We remove the Version from the expected template, as the latest env version always changes.
			delete(actualMetadata, "Version")
			// Strip new lines when comparing outputs.
			actualObj["Metadata"].(map[string]any)["Manifest"] = strings.TrimSpace(actualObj["Metadata"].(map[string]any)["Manifest"].(string))
			wantedObj["Metadata"].(map[string]any)["Manifest"] = strings.TrimSpace(wantedObj["Metadata"].(map[string]any)["Manifest"].(string))

			// THEN
			require.Equal(t, wantedObj, actualObj)
		})
	}
}
