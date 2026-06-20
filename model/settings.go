// SPDX-License-Identifier: Apache-2.0

package model

type ProjectSettings struct {
	BaseElement
	// Settings parts (polymorphic, dispatched by $Type)
	WebUI         *WebUISettings         `json:"webUI,omitempty"`
	Integration   *IntegrationSettings   `json:"integration,omitempty"`
	Configuration *ConfigurationSettings `json:"configuration,omitempty"`
	Model         *ModelSettings         `json:"model,omitempty"`
	Convention    *ConventionSettings    `json:"convention,omitempty"`
	Language      *LanguageSettings      `json:"language,omitempty"`
	Certificate   *CertificateSettings   `json:"certificate,omitempty"`
	Workflows     *WorkflowsSettings     `json:"workflows,omitempty"`
	JarDeployment *JarDeploymentSettings `json:"jarDeployment,omitempty"`
	Distribution  *DistributionSettings  `json:"distribution,omitempty"`
	// RawParts preserves the original BSON for round-trip fidelity
	RawParts []map[string]any `json:"-"`
}

type WebUISettings struct {
	BaseElement
	EnableMicroflowReachabilityAnalysis bool   `json:"enableMicroflowReachabilityAnalysis"`
	UseOptimizedClient                  string `json:"useOptimizedClient,omitempty"`
	UrlPrefix                           string `json:"urlPrefix,omitempty"`
}

type IntegrationSettings struct {
	BaseElement
}

type ConfigurationSettings struct {
	BaseElement
	Configurations []*ServerConfiguration `json:"configurations,omitempty"`
}

type ServerConfiguration struct {
	BaseElement
	Name                          string           `json:"name"`
	DatabaseType                  string           `json:"databaseType,omitempty"`
	DatabaseUrl                   string           `json:"databaseUrl,omitempty"`
	DatabaseName                  string           `json:"databaseName,omitempty"`
	DatabaseUserName              string           `json:"databaseUserName,omitempty"`
	DatabasePassword              string           `json:"databasePassword,omitempty"`
	DatabaseUseIntegratedSecurity bool             `json:"databaseUseIntegratedSecurity"`
	HttpPortNumber                int              `json:"httpPortNumber,omitempty"`
	ServerPortNumber              int              `json:"serverPortNumber,omitempty"`
	ApplicationRootUrl            string           `json:"applicationRootUrl,omitempty"`
	MaxJavaHeapSize               int              `json:"maxJavaHeapSize,omitempty"`
	ExtraJvmParameters            string           `json:"extraJvmParameters,omitempty"`
	OpenAdminPort                 bool             `json:"openAdminPort"`
	OpenHttpPort                  bool             `json:"openHttpPort"`
	ConstantValues                []*ConstantValue `json:"constantValues,omitempty"`
}

type ConstantValue struct {
	BaseElement
	ConstantId string `json:"constantId"` // Qualified name: "BusinessEvents.ServerUrl"
	Value      string `json:"value"`      // The overridden value
}

type ModelSettings struct {
	BaseElement
	AfterStartupMicroflow              string `json:"afterStartupMicroflow,omitempty"`
	BeforeShutdownMicroflow            string `json:"beforeShutdownMicroflow,omitempty"`
	HealthCheckMicroflow               string `json:"healthCheckMicroflow,omitempty"`
	AllowUserMultipleSessions          bool   `json:"allowUserMultipleSessions"`
	HashAlgorithm                      string `json:"hashAlgorithm,omitempty"`
	BcryptCost                         int    `json:"bcryptCost,omitempty"`
	JavaVersion                        string `json:"javaVersion,omitempty"`
	RoundingMode                       string `json:"roundingMode,omitempty"`
	ScheduledEventTimeZoneCode         string `json:"scheduledEventTimeZoneCode,omitempty"`
	FirstDayOfWeek                     string `json:"firstDayOfWeek,omitempty"`
	DecimalScale                       int    `json:"decimalScale,omitempty"`
	EnableDataStorageOptimisticLocking bool   `json:"enableDataStorageOptimisticLocking"`
	UseDatabaseForeignKeyConstraints   bool   `json:"useDatabaseForeignKeyConstraints"`
}

type ConventionSettings struct {
	BaseElement
	LowerCaseMicroflowVariables bool   `json:"lowerCaseMicroflowVariables"`
	DefaultAssociationStorage   string `json:"defaultAssociationStorage,omitempty"`
}

type LanguageSettings struct {
	BaseElement
	DefaultLanguageCode string     `json:"defaultLanguageCode,omitempty"`
	Languages           []Language `json:"languages,omitempty"`
}

type Language struct {
	Code                 string `json:"code"`
	CheckCompleteness    bool   `json:"checkCompleteness,omitempty"`
	CustomDateFormat     string `json:"customDateFormat,omitempty"`
	CustomDateTimeFormat string `json:"customDateTimeFormat,omitempty"`
	CustomTimeFormat     string `json:"customTimeFormat,omitempty"`
}

type CertificateSettings struct {
	BaseElement
}

type WorkflowsSettings struct {
	BaseElement
	UserEntity                string `json:"userEntity,omitempty"`
	DefaultTaskParallelism    int    `json:"defaultTaskParallelism,omitempty"`
	WorkflowEngineParallelism int    `json:"workflowEngineParallelism,omitempty"`
}

type JarDeploymentSettings struct {
	BaseElement
}

type DistributionSettings struct {
	BaseElement
	IsDistributable bool   `json:"isDistributable"`
	Version         string `json:"version,omitempty"`
}
