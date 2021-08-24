module github.com/kubermatic/telemetry-client

go 1.14

require (
	github.com/google/uuid v1.1.2
	github.com/spf13/cobra v1.1.1
	k8c.io/kubermatic/v2 v2.17.2
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.2
)

replace (
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
)
