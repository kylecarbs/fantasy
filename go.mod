module charm.land/fantasy

go 1.25.0

require (
	charm.land/x/vcr v0.1.1
	cloud.google.com/go/auth v0.18.2
	github.com/aws/aws-sdk-go-v2 v1.41.4
	github.com/aws/aws-sdk-go-v2/config v1.32.12
	github.com/aws/smithy-go v1.24.2
	github.com/charmbracelet/anthropic-sdk-go v0.0.0-20260223140439-63879b0b8dab
	github.com/charmbracelet/openai-go v0.0.0-20260319145158-d0740cc34266
	github.com/charmbracelet/x/exp/slice v0.0.0-20250904123553-b4e2667e5ad5
	github.com/charmbracelet/x/json v0.2.0
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/kaptinlin/jsonschema v0.6.10
	github.com/stretchr/testify v1.11.1
	golang.org/x/oauth2 v0.36.0
	google.golang.org/genai v1.51.0
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.9 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/kaptinlin/go-i18n v0.2.3 // indirect
	github.com/kaptinlin/jsonpointer v0.4.9 // indirect
	github.com/kaptinlin/messageformat-go v0.4.9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.3 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/api v0.264.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/dnaeon/go-vcr.v4 v4.0.6-0.20251110073552-01de4eb40290 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// coder/coder and coder/aibridge use the SasSwart perf fork of
// openai-go (deferred body serialization, appendCompact skip).
// This points to kylecarbs' copy of that fork which includes a fix
// for WithJSONSet being clobbered by deferred serialization.
// See https://github.com/kylecarbs/openai-go/pull/2
replace github.com/openai/openai-go/v3 => github.com/kylecarbs/openai-go/v3 v3.0.0-20260319113850-9477dcaedcae

// coder/coder uses a fork of charmbracelet's fork of the Anthropic Go SDK with some
// additional performance improvements. This is tracking the coder_2_33 branch.
replace github.com/charmbracelet/anthropic-sdk-go => github.com/coder/anthropic-sdk-go v0.0.0-20260415160422-a31d7d0e7067
