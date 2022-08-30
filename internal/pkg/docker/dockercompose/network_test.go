package dockercompose

import (
	"errors"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFindLinkedServices(t *testing.T) {
	nginxSvc := compose.ServiceConfig{Name: "web", Image: "nginx"}
	nginxSvcExplicitDefault := compose.ServiceConfig{
		Name:  "web",
		Image: "nginx",
		Networks: map[string]*compose.ServiceNetworkConfig{
			"default": {},
		},
	}
	postgresSvc := compose.ServiceConfig{Name: "db", Image: "postgres"}

	testCases := map[string]struct {
		inSvc       compose.ServiceConfig
		inOtherSvcs []compose.ServiceConfig

		wantLinked map[string]compose.ServiceConfig
		wantErr    error
	}{
		"no networking": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "none",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: nil,
		},
		"nginx only": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "service:web",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"web": nginxSvc,
			},
		},
		"postgres only": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "service:db",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"db": postgresSvc,
			},
		},
		"talks to self only": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "service:svc",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"svc": {
					Name:        "svc",
					Image:       "test",
					NetworkMode: "service:svc",
				},
			},
		},
		"service not found": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "service:notfound",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantErr: errors.New("no service with the name \"notfound\" found for network mode \"service:notfound\""),
		},
		"host": {
			inSvc: compose.ServiceConfig{
				Name:        "svc",
				Image:       "test",
				NetworkMode: "host",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantErr: errors.New("network mode \"host\" is not supported"),
		},
		"default network behavior": {
			inSvc: compose.ServiceConfig{
				Name:  "svc",
				Image: "test",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"svc": {
					Name:  "svc",
					Image: "test",
				},
				"web": nginxSvc,
				"db":  postgresSvc,
			},
		},
		"default network explicit in svc": {
			inSvc: compose.ServiceConfig{
				Name:  "svc",
				Image: "test",
				Networks: map[string]*compose.ServiceNetworkConfig{
					"default": {},
				},
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvc, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"svc": {
					Name:  "svc",
					Image: "test",
					Networks: map[string]*compose.ServiceNetworkConfig{
						"default": {},
					},
				},
				"web": nginxSvc,
				"db":  postgresSvc,
			},
		},
		"default network explicit in otherSvc": {
			inSvc: compose.ServiceConfig{
				Name:  "svc",
				Image: "test",
			},
			inOtherSvcs: []compose.ServiceConfig{nginxSvcExplicitDefault, postgresSvc},

			wantLinked: map[string]compose.ServiceConfig{
				"svc": {
					Name:  "svc",
					Image: "test",
				},
				"web": nginxSvcExplicitDefault,
				"db":  postgresSvc,
			},
		},
		// TODO
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			linked, err := findLinkedServices(&tc.inSvc, tc.inOtherSvcs)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantLinked, linked)
			}
		})
	}
}
