package protocol

import "encoding/json"

type AbsolutePathBuf string

type Account struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *Account) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "apiKey":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ApiKeyAccount
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgpt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "email", "planType", "type")) {
					break
				}
				var v ChatgptAccount
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "amazonBedrock":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v AmazonBedrockAccount
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ApiKeyAccount
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "email", "planType", "type") {
		var v ChatgptAccount
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v AmazonBedrockAccount
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u Account) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ApiKeyAccount struct {
	Type string `json:"type"`
}

type ChatgptAccount struct {
	Email    *string  `json:"email"`
	PlanType PlanType `json:"planType"`
	Type     string   `json:"type"`
}

func (v *ChatgptAccount) UnmarshalJSON(data []byte) error {
	type ChatgptAccountAlias struct {
		Email    *string  `json:"email"`
		PlanType PlanType `json:"planType"`
		Type     string   `json:"type"`
	}
	var aux ChatgptAccountAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	v.Email = aux.Email
	v.PlanType = aux.PlanType
	v.Type = aux.Type
	var required map[string]json.RawMessage
	if err := json.Unmarshal(data, &required); err != nil {
		return err
	}
	if _, ok := required["email"]; !ok {
		return &json.UnmarshalTypeError{Value: "missing required field", Type: nil, Field: "email"}
	}
	return nil
}

type PlanType string

const (
	PlanTypeFree                        PlanType = "free"
	PlanTypeGo                          PlanType = "go"
	PlanTypePlus                        PlanType = "plus"
	PlanTypePro                         PlanType = "pro"
	PlanTypeProlite                     PlanType = "prolite"
	PlanTypeTeam                        PlanType = "team"
	PlanTypeSelfServeBusinessProlite    PlanType = "self_serve_business_prolite"
	PlanTypeSelfServeBusinessUsageBased PlanType = "self_serve_business_usage_based"
	PlanTypeBusiness                    PlanType = "business"
	PlanTypeEnt26                       PlanType = "ent26"
	PlanTypeEnterpriseCbpAutomation     PlanType = "enterprise_cbp_automation"
	PlanTypeEnterpriseCbpUsageBased     PlanType = "enterprise_cbp_usage_based"
	PlanTypeEnterprise                  PlanType = "enterprise"
	PlanTypeEdu                         PlanType = "edu"
	PlanTypeUnknown                     PlanType = "unknown"
)

type AmazonBedrockAccount struct {
	Type                        string `json:"type"`
	UsesCodexManagedCredentials *bool  `json:"usesCodexManagedCredentials,omitempty"`
}

type AccountLoginCompletedNotification struct {
	Error                *string                      `json:"error,omitempty"`
	LoginID              *string                      `json:"loginId,omitempty"`
	OnboardingEntrypoint *DesktopOnboardingEntrypoint `json:"onboardingEntrypoint,omitempty"`
	Success              bool                         `json:"success"`
}

type DesktopOnboardingEntrypoint string

const (
	DesktopOnboardingEntrypointLifeSciences DesktopOnboardingEntrypoint = "life_sciences"
)

type AccountRateLimitsUpdatedNotification struct {
	RateLimits RateLimitSnapshot `json:"rateLimits"`
}

type RateLimitSnapshot struct {
	Credits              *CreditsSnapshot           `json:"credits,omitempty"`
	IndividualLimit      *SpendControlLimitSnapshot `json:"individualLimit,omitempty"`
	LimitID              *string                    `json:"limitId,omitempty"`
	LimitName            *string                    `json:"limitName,omitempty"`
	PlanType             *PlanType                  `json:"planType,omitempty"`
	Primary              *RateLimitWindow           `json:"primary,omitempty"`
	RateLimitReachedType *RateLimitReachedType      `json:"rateLimitReachedType,omitempty"`
	Secondary            *RateLimitWindow           `json:"secondary,omitempty"`
	SpendControlReached  *bool                      `json:"spendControlReached,omitempty"`
}

type CreditsSnapshot struct {
	Balance    *string `json:"balance,omitempty"`
	HasCredits bool    `json:"hasCredits"`
	Unlimited  bool    `json:"unlimited"`
}

type SpendControlLimitSnapshot struct {
	Limit            string `json:"limit"`
	RemainingPercent int64  `json:"remainingPercent"`
	ResetsAt         int64  `json:"resetsAt"`
	Used             string `json:"used"`
}

type RateLimitWindow struct {
	ResetsAt           *int64 `json:"resetsAt,omitempty"`
	UsedPercent        int64  `json:"usedPercent"`
	WindowDurationMins *int64 `json:"windowDurationMins,omitempty"`
}

type RateLimitReachedType string

const (
	RateLimitReachedTypeRateLimitReached                 RateLimitReachedType = "rate_limit_reached"
	RateLimitReachedTypeWorkspaceOwnerCreditsDepleted    RateLimitReachedType = "workspace_owner_credits_depleted"
	RateLimitReachedTypeWorkspaceMemberCreditsDepleted   RateLimitReachedType = "workspace_member_credits_depleted"
	RateLimitReachedTypeWorkspaceOwnerUsageLimitReached  RateLimitReachedType = "workspace_owner_usage_limit_reached"
	RateLimitReachedTypeWorkspaceMemberUsageLimitReached RateLimitReachedType = "workspace_member_usage_limit_reached"
)

type AccountTokenUsageDailyBucket struct {
	StartDate string `json:"startDate"`
	Tokens    int64  `json:"tokens"`
}

type AccountTokenUsageSummary struct {
	CurrentStreakDays     *int64 `json:"currentStreakDays,omitempty"`
	LifetimeTokens        *int64 `json:"lifetimeTokens,omitempty"`
	LongestRunningTurnSec *int64 `json:"longestRunningTurnSec,omitempty"`
	LongestStreakDays     *int64 `json:"longestStreakDays,omitempty"`
	PeakDailyTokens       *int64 `json:"peakDailyTokens,omitempty"`
}

type AccountUpdatedNotification struct {
	AuthMode *AuthMode `json:"authMode,omitempty"`
	PlanType *PlanType `json:"planType,omitempty"`
}

type AuthMode struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *AuthMode) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "apikey") {
		var v AuthModeVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "chatgpt") {
		var v AuthModeVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "chatgptAuthTokens") {
		var v AuthModeVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "headers") {
		var v AuthModeVariant4
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "agentIdentity") {
		var v AuthModeVariant5
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "personalAccessToken") {
		var v AuthModeVariant6
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "bedrockApiKey") {
		var v AuthModeVariant7
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u AuthMode) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type AuthModeVariant1 string

const (
	AuthModeVariant1Apikey AuthModeVariant1 = "apikey"
)

type AuthModeVariant2 string

const (
	AuthModeVariant2Chatgpt AuthModeVariant2 = "chatgpt"
)

type AuthModeVariant3 string

const (
	AuthModeVariant3ChatgptAuthTokens AuthModeVariant3 = "chatgptAuthTokens"
)

type AuthModeVariant4 string

const (
	AuthModeVariant4Headers AuthModeVariant4 = "headers"
)

type AuthModeVariant5 string

const (
	AuthModeVariant5AgentIdentity AuthModeVariant5 = "agentIdentity"
)

type AuthModeVariant6 string

const (
	AuthModeVariant6PersonalAccessToken AuthModeVariant6 = "personalAccessToken"
)

type AuthModeVariant7 string

const (
	AuthModeVariant7BedrockApiKey AuthModeVariant7 = "bedrockApiKey"
)

type ActivePermissionProfile struct {
	Extends *string `json:"extends,omitempty"`
	ID      string  `json:"id"`
}

type AddCreditsNudgeCreditType string

const (
	AddCreditsNudgeCreditTypeCredits    AddCreditsNudgeCreditType = "credits"
	AddCreditsNudgeCreditTypeUsageLimit AddCreditsNudgeCreditType = "usage_limit"
)

type AddCreditsNudgeEmailStatus string

const (
	AddCreditsNudgeEmailStatusSent           AddCreditsNudgeEmailStatus = "sent"
	AddCreditsNudgeEmailStatusCooldownActive AddCreditsNudgeEmailStatus = "cooldown_active"
)

type AdditionalContextEntry struct {
	Kind  AdditionalContextKind `json:"kind"`
	Value string                `json:"value"`
}

type AdditionalContextKind string

const (
	AdditionalContextKindUntrusted   AdditionalContextKind = "untrusted"
	AdditionalContextKindApplication AdditionalContextKind = "application"
)

type AdditionalFileSystemPermissions struct {
	Entries          []FileSystemSandboxEntry `json:"entries,omitempty"`
	GlobScanMaxDepth *int64                   `json:"globScanMaxDepth,omitempty"`
	Read             []LegacyAppPathString    `json:"read,omitempty"`
	Write            []LegacyAppPathString    `json:"write,omitempty"`
}

type FileSystemSandboxEntry struct {
	Access FileSystemAccessMode `json:"access"`
	Path   FileSystemPath       `json:"path"`
}

type FileSystemAccessMode string

const (
	FileSystemAccessModeRead  FileSystemAccessMode = "read"
	FileSystemAccessModeWrite FileSystemAccessMode = "write"
	FileSystemAccessModeDeny  FileSystemAccessMode = "deny"
)

type FileSystemPath struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FileSystemPath) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "path":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "path", "type")) {
					break
				}
				var v PathFileSystemPath
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "glob_pattern":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "pattern", "type")) {
					break
				}
				var v GlobPatternFileSystemPath
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "special":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type", "value")) {
					break
				}
				var v SpecialFileSystemPath
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "path", "type") {
		var v PathFileSystemPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "pattern", "type") {
		var v GlobPatternFileSystemPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type", "value") {
		var v SpecialFileSystemPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FileSystemPath) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type PathFileSystemPath struct {
	Path LegacyAppPathString `json:"path"`
	Type string              `json:"type"`
}

type LegacyAppPathString string

type GlobPatternFileSystemPath struct {
	Pattern string `json:"pattern"`
	Type    string `json:"type"`
}

type SpecialFileSystemPath struct {
	Type  string                `json:"type"`
	Value FileSystemSpecialPath `json:"value"`
}

type FileSystemSpecialPath struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FileSystemSpecialPath) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksObject(data) && jsonHasKeys(data, "kind") {
		var v RootFileSystemSpecialPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "kind") {
		var v MinimalFileSystemSpecialPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "kind") {
		var v KindFileSystemSpecialPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "kind") {
		var v TmpdirFileSystemSpecialPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "kind") {
		var v SlashTmpFileSystemSpecialPath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "kind", "path") {
		var v FileSystemSpecialPathVariant6
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FileSystemSpecialPath) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type RootFileSystemSpecialPath struct {
	Kind string `json:"kind"`
}

type MinimalFileSystemSpecialPath struct {
	Kind string `json:"kind"`
}

type KindFileSystemSpecialPath struct {
	Kind    string               `json:"kind"`
	Subpath *LegacyAppPathString `json:"subpath,omitempty"`
}

type TmpdirFileSystemSpecialPath struct {
	Kind string `json:"kind"`
}

type SlashTmpFileSystemSpecialPath struct {
	Kind string `json:"kind"`
}

type FileSystemSpecialPathVariant6 struct {
	Kind    string               `json:"kind"`
	Path    string               `json:"path"`
	Subpath *LegacyAppPathString `json:"subpath,omitempty"`
}

type AdditionalNetworkPermissions struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type AdditionalPermissionProfile struct {
	FileSystem *AdditionalFileSystemPermissions `json:"fileSystem,omitempty"`
	Network    *AdditionalNetworkPermissions    `json:"network,omitempty"`
}

type AgentMessageDeltaNotification struct {
	Delta    string `json:"delta"`
	ItemID   string `json:"itemId"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type AgentMessageInputContent struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *AgentMessageInputContent) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "input_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v InputTextAgentMessageInputContent
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "encrypted_content":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type")) {
					break
				}
				var v EncryptedContentAgentMessageInputContent
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v InputTextAgentMessageInputContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type") {
		var v EncryptedContentAgentMessageInputContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u AgentMessageInputContent) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InputTextAgentMessageInputContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type EncryptedContentAgentMessageInputContent struct {
	EncryptedContent string `json:"encrypted_content"`
	Type             string `json:"type"`
}

type AgentPath string

type AnalyticsConfig struct {
	Enabled *bool                      `json:"enabled,omitempty"`
	Extras  map[string]json.RawMessage `json:"-"`
}

func (v *AnalyticsConfig) UnmarshalJSON(data []byte) error {
	type AnalyticsConfigAlias struct {
		Enabled *bool `json:"enabled,omitempty"`
	}
	var aux AnalyticsConfigAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	v.Enabled = aux.Enabled
	var extras map[string]json.RawMessage
	if err := json.Unmarshal(data, &extras); err != nil {
		return err
	}
	delete(extras, "enabled")
	if len(extras) == 0 {
		v.Extras = nil
	} else {
		v.Extras = extras
	}
	return nil
}

func (v AnalyticsConfig) MarshalJSON() ([]byte, error) {
	type AnalyticsConfigAlias struct {
		Enabled *bool `json:"enabled,omitempty"`
	}
	aux := AnalyticsConfigAlias{
		Enabled: v.Enabled,
	}
	encoded, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, err
	}
	for key, value := range v.Extras {
		if _, exists := payload[key]; exists {
			continue
		}
		payload[key] = append(json.RawMessage(nil), value...)
	}
	return json.Marshal(payload)
}

type AppBranding struct {
	Category          *string `json:"category,omitempty"`
	Developer         *string `json:"developer,omitempty"`
	IsDiscoverableApp bool    `json:"isDiscoverableApp"`
	PrivacyPolicy     *string `json:"privacyPolicy,omitempty"`
	TermsOfService    *string `json:"termsOfService,omitempty"`
	Website           *string `json:"website,omitempty"`
}

type AppConfig struct {
	ApprovalsReviewer        *ApprovalsReviewer `json:"approvals_reviewer,omitempty"`
	DefaultToolsApprovalMode *AppToolApproval   `json:"default_tools_approval_mode,omitempty"`
	DefaultToolsEnabled      *bool              `json:"default_tools_enabled,omitempty"`
	DestructiveEnabled       *bool              `json:"destructive_enabled,omitempty"`
	Enabled                  *bool              `json:"enabled,omitempty"`
	OpenWorldEnabled         *bool              `json:"open_world_enabled,omitempty"`
	Tools                    *AppToolsConfig    `json:"tools,omitempty"`
}

type ApprovalsReviewer string

const (
	ApprovalsReviewerUser             ApprovalsReviewer = "user"
	ApprovalsReviewerAutoReview       ApprovalsReviewer = "auto_review"
	ApprovalsReviewerGuardianSubagent ApprovalsReviewer = "guardian_subagent"
)

type AppToolApproval string

const (
	AppToolApprovalAuto    AppToolApproval = "auto"
	AppToolApprovalPrompt  AppToolApproval = "prompt"
	AppToolApprovalWrites  AppToolApproval = "writes"
	AppToolApprovalApprove AppToolApproval = "approve"
)

type AppToolsConfig struct {
}

type AppInfo struct {
	AppMetadata         *AppMetadata      `json:"appMetadata,omitempty"`
	Branding            *AppBranding      `json:"branding,omitempty"`
	Description         *string           `json:"description,omitempty"`
	DistributionChannel *string           `json:"distributionChannel,omitempty"`
	IconAssets          map[string]string `json:"iconAssets,omitempty"`
	IconDarkAssets      map[string]string `json:"iconDarkAssets,omitempty"`
	ID                  string            `json:"id"`
	InstallUrl          *string           `json:"installUrl,omitempty"`
	IsAccessible        *bool             `json:"isAccessible,omitempty"`
	IsEnabled           *bool             `json:"isEnabled,omitempty"`
	Labels              map[string]string `json:"labels,omitempty"`
	LogoUrl             *string           `json:"logoUrl,omitempty"`
	LogoUrlDark         *string           `json:"logoUrlDark,omitempty"`
	Name                string            `json:"name"`
	PluginDisplayNames  []string          `json:"pluginDisplayNames,omitempty"`
}

type AppMetadata struct {
	Categories                 []string        `json:"categories,omitempty"`
	Developer                  *string         `json:"developer,omitempty"`
	FirstPartyRequiresInstall  *bool           `json:"firstPartyRequiresInstall,omitempty"`
	Review                     *AppReview      `json:"review,omitempty"`
	Screenshots                []AppScreenshot `json:"screenshots,omitempty"`
	SeoDescription             *string         `json:"seoDescription,omitempty"`
	ShowInComposerWhenUnlinked *bool           `json:"showInComposerWhenUnlinked,omitempty"`
	SubCategories              []string        `json:"subCategories,omitempty"`
	Version                    *string         `json:"version,omitempty"`
	VersionID                  *string         `json:"versionId,omitempty"`
	VersionNotes               *string         `json:"versionNotes,omitempty"`
}

type AppReview struct {
	Status string `json:"status"`
}

type AppScreenshot struct {
	FileID     *string `json:"fileId,omitempty"`
	Url        *string `json:"url,omitempty"`
	UserPrompt string  `json:"userPrompt"`
}

type AppListUpdatedNotification struct {
	Data []AppInfo `json:"data"`
}

type AppSummary struct {
	Category    *string `json:"category,omitempty"`
	Description *string `json:"description,omitempty"`
	ID          string  `json:"id"`
	InstallUrl  *string `json:"installUrl,omitempty"`
	Name        string  `json:"name"`
}

type AppTemplateSummary struct {
	CanonicalConnectorID *string                       `json:"canonicalConnectorId,omitempty"`
	Category             *string                       `json:"category,omitempty"`
	Description          *string                       `json:"description,omitempty"`
	LogoUrl              *string                       `json:"logoUrl,omitempty"`
	LogoUrlDark          *string                       `json:"logoUrlDark,omitempty"`
	MaterializedAppIds   []string                      `json:"materializedAppIds"`
	Name                 string                        `json:"name"`
	Reason               *AppTemplateUnavailableReason `json:"reason,omitempty"`
	TemplateID           string                        `json:"templateId"`
}

type AppTemplateUnavailableReason string

const (
	AppTemplateUnavailableReasonNOTCONFIGUREDFORWORKSPACE AppTemplateUnavailableReason = "NOT_CONFIGURED_FOR_WORKSPACE"
	AppTemplateUnavailableReasonNOACTIVEWORKSPACE         AppTemplateUnavailableReason = "NO_ACTIVE_WORKSPACE"
)

type AppToolConfig struct {
	ApprovalMode *AppToolApproval `json:"approval_mode,omitempty"`
	Enabled      *bool            `json:"enabled,omitempty"`
}

type AppToolSummary struct {
	Description    string  `json:"description"`
	DisabledReason *string `json:"disabledReason,omitempty"`
	IsEnabled      *bool   `json:"isEnabled,omitempty"`
	IsReadOnly     *bool   `json:"isReadOnly,omitempty"`
	Name           string  `json:"name"`
	Title          *string `json:"title,omitempty"`
}

type ApplyPatchApprovalParams struct {
	CallID         string                `json:"callId"`
	ConversationID ThreadId              `json:"conversationId"`
	FileChanges    map[string]FileChange `json:"fileChanges"`
	GrantRoot      *string               `json:"grantRoot,omitempty"`
	Reason         *string               `json:"reason,omitempty"`
}

type ThreadId string

type FileChange struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FileChange) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "add":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "content", "type")) {
					break
				}
				var v AddFileChange
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "delete":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "content", "type")) {
					break
				}
				var v DeleteFileChange
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "update":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type", "unified_diff")) {
					break
				}
				var v UpdateFileChange
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "content", "type") {
		var v AddFileChange
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "content", "type") {
		var v DeleteFileChange
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type", "unified_diff") {
		var v UpdateFileChange
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FileChange) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type AddFileChange struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

type DeleteFileChange struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

type UpdateFileChange struct {
	MovePath    *string `json:"move_path,omitempty"`
	Type        string  `json:"type"`
	UnifiedDiff string  `json:"unified_diff"`
}

type ApplyPatchApprovalResponse struct {
	Decision ReviewDecision `json:"decision"`
}

type ReviewDecision struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ReviewDecision) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "approved") {
		var v ReviewDecisionVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "approved_execpolicy_amendment") {
		var v ApprovedExecpolicyAmendmentReviewDecision
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "approved_for_session") {
		var v ReviewDecisionVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "network_policy_amendment") {
		var v NetworkPolicyAmendmentReviewDecision
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "denied") {
		var v DeniedReviewDecision
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "timed_out") {
		var v ReviewDecisionVariant6
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "abort") {
		var v ReviewDecisionVariant7
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ReviewDecision) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ReviewDecisionVariant1 string

const (
	ReviewDecisionVariant1Approved ReviewDecisionVariant1 = "approved"
)

type ApprovedExecpolicyAmendmentReviewDecision struct {
	ApprovedExecpolicyAmendment map[string]any `json:"approved_execpolicy_amendment"`
}

type ReviewDecisionVariant3 string

const (
	ReviewDecisionVariant3ApprovedForSession ReviewDecisionVariant3 = "approved_for_session"
)

type NetworkPolicyAmendmentReviewDecision struct {
	NetworkPolicyAmendment map[string]any `json:"network_policy_amendment"`
}

type DeniedReviewDecision struct {
	Denied map[string]any `json:"denied"`
}

type ReviewDecisionVariant6 string

const (
	ReviewDecisionVariant6TimedOut ReviewDecisionVariant6 = "timed_out"
)

type ReviewDecisionVariant7 string

const (
	ReviewDecisionVariant7Abort ReviewDecisionVariant7 = "abort"
)

type AppsConfig struct {
	Default *AppsDefaultConfig `json:"_default,omitempty"`
}

type AppsDefaultConfig struct {
	ApprovalsReviewer        *ApprovalsReviewer `json:"approvals_reviewer,omitempty"`
	DefaultToolsApprovalMode *AppToolApproval   `json:"default_tools_approval_mode,omitempty"`
	DestructiveEnabled       *bool              `json:"destructive_enabled,omitempty"`
	Enabled                  *bool              `json:"enabled,omitempty"`
	OpenWorldEnabled         *bool              `json:"open_world_enabled,omitempty"`
}

type AppsInstalledParams struct {
	ForceRefresh *bool   `json:"forceRefresh,omitempty"`
	ThreadID     *string `json:"threadId,omitempty"`
}

type AppsInstalledResponse struct {
	Apps []InstalledApp `json:"apps"`
}

type InstalledApp struct {
	Callable    bool    `json:"callable"`
	Enabled     bool    `json:"enabled"`
	ID          string  `json:"id"`
	RuntimeName *string `json:"runtimeName,omitempty"`
}

type AppsListParams struct {
	Cursor       *string `json:"cursor,omitempty"`
	ForceRefetch *bool   `json:"forceRefetch,omitempty"`
	Limit        *int64  `json:"limit,omitempty"`
	ThreadID     *string `json:"threadId,omitempty"`
}

type AppsListResponse struct {
	Data       []AppInfo `json:"data"`
	NextCursor *string   `json:"nextCursor,omitempty"`
}

type AppsReadParams struct {
	AppIds       []string `json:"appIds"`
	IncludeTools *bool    `json:"includeTools,omitempty"`
}

type AppsReadResponse struct {
	Apps          []ConnectorMetadata `json:"apps"`
	MissingAppIds []string            `json:"missingAppIds"`
}

type ConnectorMetadata struct {
	Description         *string          `json:"description,omitempty"`
	DistributionChannel *string          `json:"distributionChannel,omitempty"`
	IconUrl             *string          `json:"iconUrl,omitempty"`
	IconUrlDark         *string          `json:"iconUrlDark,omitempty"`
	ID                  string           `json:"id"`
	InstallUrl          *string          `json:"installUrl,omitempty"`
	Name                string           `json:"name"`
	PluginDisplayNames  []string         `json:"pluginDisplayNames,omitempty"`
	ToolSummaries       []AppToolSummary `json:"toolSummaries,omitempty"`
}

type AskForApproval struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *AskForApproval) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "untrusted", "on-request", "never") {
		var v AskForApprovalVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "granular") {
		var v GranularAskForApproval
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u AskForApproval) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type AskForApprovalVariant1 string

const (
	AskForApprovalVariant1Untrusted AskForApprovalVariant1 = "untrusted"
	AskForApprovalVariant1OnRequest AskForApprovalVariant1 = "on-request"
	AskForApprovalVariant1Never     AskForApprovalVariant1 = "never"
)

type GranularAskForApproval struct {
	Granular map[string]any `json:"granular"`
}

type AttestationGenerateParams struct {
}

type AttestationGenerateResponse struct {
	Token string `json:"token"`
}

type AutoCompactTokenLimitScope struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *AutoCompactTokenLimitScope) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "total") {
		var v AutoCompactTokenLimitScopeVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "body_after_prefix") {
		var v AutoCompactTokenLimitScopeVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u AutoCompactTokenLimitScope) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type AutoCompactTokenLimitScopeVariant1 string

const (
	AutoCompactTokenLimitScopeVariant1Total AutoCompactTokenLimitScopeVariant1 = "total"
)

type AutoCompactTokenLimitScopeVariant2 string

const (
	AutoCompactTokenLimitScopeVariant2BodyAfterPrefix AutoCompactTokenLimitScopeVariant2 = "body_after_prefix"
)

type AutoReviewDecisionSource string

const (
	AutoReviewDecisionSourceAgent AutoReviewDecisionSource = "agent"
)

type BrowserUseRequirements struct {
	DisableAutoReview *bool `json:"disableAutoReview,omitempty"`
}

type ByteRange struct {
	End   int64 `json:"end"`
	Start int64 `json:"start"`
}

type CancelLoginAccountParams struct {
	LoginID string `json:"loginId"`
}

type CancelLoginAccountResponse struct {
	Status CancelLoginAccountStatus `json:"status"`
}

type CancelLoginAccountStatus string

const (
	CancelLoginAccountStatusCanceled CancelLoginAccountStatus = "canceled"
	CancelLoginAccountStatusNotFound CancelLoginAccountStatus = "notFound"
)

type CapabilityRootLocation struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *CapabilityRootLocation) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "environment":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "environmentId", "path", "type")) {
					break
				}
				var v EnvironmentCapabilityRootLocation
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "environmentId", "path", "type") {
		var v EnvironmentCapabilityRootLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u CapabilityRootLocation) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type EnvironmentCapabilityRootLocation struct {
	EnvironmentID string `json:"environmentId"`
	Path          string `json:"path"`
	Type          string `json:"type"`
}

type ChatgptAuthTokensRefreshParams struct {
	PreviousAccountID *string                        `json:"previousAccountId,omitempty"`
	Reason            ChatgptAuthTokensRefreshReason `json:"reason"`
}

type ChatgptAuthTokensRefreshReason struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ChatgptAuthTokensRefreshReason) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "unauthorized") {
		var v ChatgptAuthTokensRefreshReasonVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ChatgptAuthTokensRefreshReason) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ChatgptAuthTokensRefreshReasonVariant1 string

const (
	ChatgptAuthTokensRefreshReasonVariant1Unauthorized ChatgptAuthTokensRefreshReasonVariant1 = "unauthorized"
)

type ChatgptAuthTokensRefreshResponse struct {
	AccessToken      string  `json:"accessToken"`
	ChatgptAccountID string  `json:"chatgptAccountId"`
	ChatgptPlanType  *string `json:"chatgptPlanType,omitempty"`
}

type ClientInfo struct {
	Name    string  `json:"name"`
	Title   *string `json:"title,omitempty"`
	Version string  `json:"version"`
}

type ClientNotification struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ClientNotification) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "method") {
		if disc, ok := unionDiscriminator(data, "method"); ok {
			switch disc {
			case "initialized":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method")) {
					break
				}
				var v InitializedNotification
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method") {
		var v InitializedNotification
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ClientNotification) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InitializedNotification struct {
	Method string `json:"method"`
}

type ClientRequest struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ClientRequest) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "method") {
		if disc, ok := unionDiscriminator(data, "method"); ok {
			switch disc {
			case "initialize":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v InitializeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/start":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/resume":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadResumeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/fork":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadForkRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/archive":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadArchiveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/delete":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadDeleteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/unsubscribe":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadUnsubscribeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/name/set":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadNameSetRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/goal/set":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadGoalSetRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/goal/get":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadGoalGetRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/goal/clear":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadGoalClearRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/metadata/update":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadMetadataUpdateRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/section/move":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadSectionMoveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/unarchive":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadUnarchiveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/compact/start":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadCompactStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/shellCommand":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadShellCommandRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/approveGuardianDeniedAction":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadApproveGuardianDeniedActionRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/rollback":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadRollbackRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "threadSection/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadSectionListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "threadSection/create":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadSectionCreateRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "threadSection/update":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadSectionUpdateRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "threadSection/delete":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadSectionDeleteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/loaded/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadLoadedListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/inject_items":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ThreadInjectItemsRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "skills/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v SkillsListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "skills/extraRoots/set":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v SkillsExtraRootsSetRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "hooks/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v HooksListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "marketplace/add":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v MarketplaceAddRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "marketplace/remove":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v MarketplaceRemoveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "marketplace/upgrade":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v MarketplaceUpgradeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/installed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginInstalledRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/skill/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginSkillReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/share/save":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginShareSaveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/share/updateTargets":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginShareUpdateTargetsRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/share/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginShareListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/share/checkout":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginShareCheckoutRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/share/delete":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginShareDeleteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "app/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AppReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "app/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AppListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "app/installed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AppInstalledRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/readFile":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsReadFileRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/writeFile":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsWriteFileRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/createDirectory":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsCreateDirectoryRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/getMetadata":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsGetMetadataRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/readDirectory":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsReadDirectoryRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/remove":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsRemoveRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/copy":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsCopyRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/watch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsWatchRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/unwatch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FsUnwatchRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "skills/config/write":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v SkillsConfigWriteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/install":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginInstallRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plugin/uninstall":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PluginUninstallRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/start":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v TurnStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/steer":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v TurnSteerRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/interrupt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v TurnInterruptRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "review/start":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ReviewStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "model/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ModelListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "modelProvider/capabilities/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ModelProviderCapabilitiesReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "experimentalFeature/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExperimentalFeatureListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "permissionProfile/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v PermissionProfileListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "experimentalFeature/enablement/set":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExperimentalFeatureEnablementSetRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/oauth/login":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v McpServerOauthLoginRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "config/mcpServer/reload":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v ConfigMcpServerReloadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServerStatus/list":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v McpServerStatusListRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/resource/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v McpServerResourceReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/tool/call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v McpServerToolCallRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "windowsSandbox/setupStart":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v WindowsSandboxSetupStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "windowsSandbox/readiness":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v WindowsSandboxReadinessRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/login/start":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountLoginStartRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/login/cancel":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountLoginCancelRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/logout":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v AccountLogoutRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/rateLimits/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v AccountRateLimitsReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/rateLimitResetCredit/consume":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountRateLimitResetCreditConsumeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/usage/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v AccountUsageReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/workspaceMessages/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v AccountWorkspaceMessagesReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/sendAddCreditsNudgeEmail":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountSendAddCreditsNudgeEmailRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "feedback/upload":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FeedbackUploadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "command/exec":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v CommandExecRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "command/exec/write":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v CommandExecWriteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "command/exec/terminate":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v CommandExecTerminateRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "command/exec/resize":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v CommandExecResizeRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "config/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ConfigReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/detect":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExternalAgentConfigDetectRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/import":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExternalAgentConfigImportRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/import/recordHistory":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExternalAgentConfigImportRecordHistoryRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/import/readHistories":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v ExternalAgentConfigImportReadHistoriesRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "config/value/write":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ConfigValueWriteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "config/batchWrite":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ConfigBatchWriteRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "configRequirements/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method")) {
					break
				}
				var v ConfigRequirementsReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountReadRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fuzzyFileSearch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v FuzzyFileSearchRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v InitializeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadResumeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadForkRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadArchiveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadDeleteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadUnsubscribeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadNameSetRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadGoalSetRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadGoalGetRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadGoalClearRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadMetadataUpdateRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadSectionMoveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadUnarchiveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadCompactStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadShellCommandRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadApproveGuardianDeniedActionRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadRollbackRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadSectionListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadSectionCreateRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadSectionUpdateRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadSectionDeleteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadLoadedListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ThreadInjectItemsRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v SkillsListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v SkillsExtraRootsSetRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v HooksListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v MarketplaceAddRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v MarketplaceRemoveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v MarketplaceUpgradeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginInstalledRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginSkillReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginShareSaveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginShareUpdateTargetsRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginShareListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginShareCheckoutRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginShareDeleteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AppReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AppListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AppInstalledRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsReadFileRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsWriteFileRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsCreateDirectoryRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsGetMetadataRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsReadDirectoryRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsRemoveRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsCopyRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsWatchRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FsUnwatchRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v SkillsConfigWriteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginInstallRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PluginUninstallRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v TurnStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v TurnSteerRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v TurnInterruptRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ReviewStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ModelListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ModelProviderCapabilitiesReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExperimentalFeatureListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v PermissionProfileListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExperimentalFeatureEnablementSetRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v McpServerOauthLoginRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v ConfigMcpServerReloadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v McpServerStatusListRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v McpServerResourceReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v McpServerToolCallRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v WindowsSandboxSetupStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v WindowsSandboxReadinessRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountLoginStartRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountLoginCancelRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v AccountLogoutRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v AccountRateLimitsReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountRateLimitResetCreditConsumeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v AccountUsageReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v AccountWorkspaceMessagesReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountSendAddCreditsNudgeEmailRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FeedbackUploadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v CommandExecRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v CommandExecWriteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v CommandExecTerminateRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v CommandExecResizeRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ConfigReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExternalAgentConfigDetectRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExternalAgentConfigImportRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExternalAgentConfigImportRecordHistoryRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v ExternalAgentConfigImportReadHistoriesRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ConfigValueWriteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ConfigBatchWriteRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method") {
		var v ConfigRequirementsReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountReadRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v FuzzyFileSearchRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ClientRequest) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InitializeRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params InitializeParams `json:"params"`
}

type RequestId struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *RequestId) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksString(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksNumber(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u RequestId) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InitializeParams struct {
	Capabilities *InitializeCapabilities `json:"capabilities,omitempty"`
	ClientInfo   ClientInfo              `json:"clientInfo"`
}

type InitializeCapabilities struct {
	ExperimentalApi                *bool    `json:"experimentalApi,omitempty"`
	McpServerOpenaiFormElicitation *bool    `json:"mcpServerOpenaiFormElicitation,omitempty"`
	OptOutNotificationMethods      []string `json:"optOutNotificationMethods,omitempty"`
	RequestAttestation             *bool    `json:"requestAttestation,omitempty"`
}

type ThreadStartRequestEnvelope struct {
	ID     RequestId         `json:"id"`
	Method string            `json:"method"`
	Params ThreadStartParams `json:"params"`
}

type ThreadStartParams struct {
	ApprovalPolicy        *AskForApproval    `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *ApprovalsReviewer `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string            `json:"baseInstructions,omitempty"`
	Config                map[string]any     `json:"config,omitempty"`
	Cwd                   *string            `json:"cwd,omitempty"`
	DeveloperInstructions *string            `json:"developerInstructions,omitempty"`
	Ephemeral             *bool              `json:"ephemeral,omitempty"`
	Model                 *string            `json:"model,omitempty"`
	ModelProvider         *string            `json:"modelProvider,omitempty"`
	Personality           *Personality       `json:"personality,omitempty"`
	Sandbox               *SandboxMode       `json:"sandbox,omitempty"`
	ServiceName           *string            `json:"serviceName,omitempty"`
	ServiceTier           *string            `json:"serviceTier,omitempty"`
	SessionStartSource    *ThreadStartSource `json:"sessionStartSource,omitempty"`
	ThreadSource          *ThreadSource      `json:"threadSource,omitempty"`
}

type Personality string

const (
	PersonalityNone      Personality = "none"
	PersonalityFriendly  Personality = "friendly"
	PersonalityPragmatic Personality = "pragmatic"
)

type SandboxMode string

const (
	SandboxModeReadOnly         SandboxMode = "read-only"
	SandboxModeWorkspaceWrite   SandboxMode = "workspace-write"
	SandboxModeDangerFullAccess SandboxMode = "danger-full-access"
)

type ThreadStartSource string

const (
	ThreadStartSourceStartup ThreadStartSource = "startup"
	ThreadStartSourceClear   ThreadStartSource = "clear"
)

type ThreadSource string

type ThreadResumeRequestEnvelope struct {
	ID     RequestId          `json:"id"`
	Method string             `json:"method"`
	Params ThreadResumeParams `json:"params"`
}

type ThreadResumeParams struct {
	ApprovalPolicy        *AskForApproval    `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *ApprovalsReviewer `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string            `json:"baseInstructions,omitempty"`
	Config                map[string]any     `json:"config,omitempty"`
	Cwd                   *string            `json:"cwd,omitempty"`
	DeveloperInstructions *string            `json:"developerInstructions,omitempty"`
	Model                 *string            `json:"model,omitempty"`
	ModelProvider         *string            `json:"modelProvider,omitempty"`
	Personality           *Personality       `json:"personality,omitempty"`
	Sandbox               *SandboxMode       `json:"sandbox,omitempty"`
	ServiceTier           *string            `json:"serviceTier,omitempty"`
	ThreadID              string             `json:"threadId"`
}

type ThreadForkRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params ThreadForkParams `json:"params"`
}

type ThreadForkParams struct {
	ApprovalPolicy        *AskForApproval    `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *ApprovalsReviewer `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string            `json:"baseInstructions,omitempty"`
	Config                map[string]any     `json:"config,omitempty"`
	Cwd                   *string            `json:"cwd,omitempty"`
	DeveloperInstructions *string            `json:"developerInstructions,omitempty"`
	Ephemeral             *bool              `json:"ephemeral,omitempty"`
	LastTurnID            *string            `json:"lastTurnId,omitempty"`
	Model                 *string            `json:"model,omitempty"`
	ModelProvider         *string            `json:"modelProvider,omitempty"`
	Sandbox               *SandboxMode       `json:"sandbox,omitempty"`
	ServiceTier           *string            `json:"serviceTier,omitempty"`
	ThreadID              string             `json:"threadId"`
	ThreadSource          *ThreadSource      `json:"threadSource,omitempty"`
}

type ThreadArchiveRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params ThreadArchiveParams `json:"params"`
}

type ThreadArchiveParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadDeleteRequestEnvelope struct {
	ID     RequestId          `json:"id"`
	Method string             `json:"method"`
	Params ThreadDeleteParams `json:"params"`
}

type ThreadDeleteParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadUnsubscribeRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params ThreadUnsubscribeParams `json:"params"`
}

type ThreadUnsubscribeParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadNameSetRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params ThreadSetNameParams `json:"params"`
}

type ThreadSetNameParams struct {
	Name     string `json:"name"`
	ThreadID string `json:"threadId"`
}

type ThreadGoalSetRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params ThreadGoalSetParams `json:"params"`
}

type ThreadGoalSetParams struct {
	Objective   *string           `json:"objective,omitempty"`
	Status      *ThreadGoalStatus `json:"status,omitempty"`
	ThreadID    string            `json:"threadId"`
	TokenBudget *int64            `json:"tokenBudget,omitempty"`
}

type ThreadGoalStatus string

const (
	ThreadGoalStatusActive        ThreadGoalStatus = "active"
	ThreadGoalStatusPaused        ThreadGoalStatus = "paused"
	ThreadGoalStatusBlocked       ThreadGoalStatus = "blocked"
	ThreadGoalStatusUsageLimited  ThreadGoalStatus = "usageLimited"
	ThreadGoalStatusBudgetLimited ThreadGoalStatus = "budgetLimited"
	ThreadGoalStatusComplete      ThreadGoalStatus = "complete"
)

type ThreadGoalGetRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params ThreadGoalGetParams `json:"params"`
}

type ThreadGoalGetParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadGoalClearRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params ThreadGoalClearParams `json:"params"`
}

type ThreadGoalClearParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadMetadataUpdateRequestEnvelope struct {
	ID     RequestId                  `json:"id"`
	Method string                     `json:"method"`
	Params ThreadMetadataUpdateParams `json:"params"`
}

type ThreadMetadataUpdateParams struct {
	GitInfo  *ThreadMetadataGitInfoUpdateParams `json:"gitInfo,omitempty"`
	ThreadID string                             `json:"threadId"`
}

type ThreadMetadataGitInfoUpdateParams struct {
	Branch    *string `json:"branch,omitempty"`
	OriginUrl *string `json:"originUrl,omitempty"`
	Sha       *string `json:"sha,omitempty"`
}

type ThreadSectionMoveRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params ThreadSectionMoveParams `json:"params"`
}

type ThreadSectionMoveParams struct {
	BeforeThreadID *string `json:"beforeThreadId,omitempty"`
	SectionID      *string `json:"sectionId"`
	ThreadID       string  `json:"threadId"`
}

func (v *ThreadSectionMoveParams) UnmarshalJSON(data []byte) error {
	type ThreadSectionMoveParamsAlias struct {
		BeforeThreadID *string `json:"beforeThreadId,omitempty"`
		SectionID      *string `json:"sectionId"`
		ThreadID       string  `json:"threadId"`
	}
	var aux ThreadSectionMoveParamsAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	v.BeforeThreadID = aux.BeforeThreadID
	v.SectionID = aux.SectionID
	v.ThreadID = aux.ThreadID
	var required map[string]json.RawMessage
	if err := json.Unmarshal(data, &required); err != nil {
		return err
	}
	if _, ok := required["sectionId"]; !ok {
		return &json.UnmarshalTypeError{Value: "missing required field", Type: nil, Field: "sectionId"}
	}
	return nil
}

type ThreadUnarchiveRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params ThreadUnarchiveParams `json:"params"`
}

type ThreadUnarchiveParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadCompactStartRequestEnvelope struct {
	ID     RequestId                `json:"id"`
	Method string                   `json:"method"`
	Params ThreadCompactStartParams `json:"params"`
}

type ThreadCompactStartParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadShellCommandRequestEnvelope struct {
	ID     RequestId                `json:"id"`
	Method string                   `json:"method"`
	Params ThreadShellCommandParams `json:"params"`
}

type ThreadShellCommandParams struct {
	Command  string `json:"command"`
	ThreadID string `json:"threadId"`
}

type ThreadApproveGuardianDeniedActionRequestEnvelope struct {
	ID     RequestId                               `json:"id"`
	Method string                                  `json:"method"`
	Params ThreadApproveGuardianDeniedActionParams `json:"params"`
}

type ThreadApproveGuardianDeniedActionParams struct {
	Event    any    `json:"event"`
	ThreadID string `json:"threadId"`
}

type ThreadRollbackRequestEnvelope struct {
	ID     RequestId            `json:"id"`
	Method string               `json:"method"`
	Params ThreadRollbackParams `json:"params"`
}

type ThreadRollbackParams struct {
	NumTurns int64  `json:"numTurns"`
	ThreadID string `json:"threadId"`
}

type ThreadListRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params ThreadListParams `json:"params"`
}

type ThreadListParams struct {
	Archived       *bool                `json:"archived,omitempty"`
	Cursor         *string              `json:"cursor,omitempty"`
	Cwd            *ThreadListCwdFilter `json:"cwd,omitempty"`
	Limit          *int64               `json:"limit,omitempty"`
	ModelProviders []string             `json:"modelProviders,omitempty"`
	SearchTerm     *string              `json:"searchTerm,omitempty"`
	SectionID      *string              `json:"sectionId,omitempty"`
	SortDirection  *SortDirection       `json:"sortDirection,omitempty"`
	SortKey        *ThreadSortKey       `json:"sortKey,omitempty"`
	SourceKinds    []ThreadSourceKind   `json:"sourceKinds,omitempty"`
	UseStateDbOnly *bool                `json:"useStateDbOnly,omitempty"`
}

type ThreadListCwdFilter struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ThreadListCwdFilter) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksString(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksArray(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ThreadListCwdFilter) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

type ThreadSortKey string

const (
	ThreadSortKeyCreatedAt       ThreadSortKey = "created_at"
	ThreadSortKeyUpdatedAt       ThreadSortKey = "updated_at"
	ThreadSortKeyRecencyAt       ThreadSortKey = "recency_at"
	ThreadSortKeySectionPosition ThreadSortKey = "section_position"
)

type ThreadSourceKind string

const (
	ThreadSourceKindCli                 ThreadSourceKind = "cli"
	ThreadSourceKindVscode              ThreadSourceKind = "vscode"
	ThreadSourceKindExec                ThreadSourceKind = "exec"
	ThreadSourceKindAppServer           ThreadSourceKind = "appServer"
	ThreadSourceKindSubAgent            ThreadSourceKind = "subAgent"
	ThreadSourceKindSubAgentReview      ThreadSourceKind = "subAgentReview"
	ThreadSourceKindSubAgentCompact     ThreadSourceKind = "subAgentCompact"
	ThreadSourceKindSubAgentThreadSpawn ThreadSourceKind = "subAgentThreadSpawn"
	ThreadSourceKindSubAgentOther       ThreadSourceKind = "subAgentOther"
	ThreadSourceKindUnknown             ThreadSourceKind = "unknown"
)

type ThreadSectionListRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params ThreadSectionListParams `json:"params"`
}

type ThreadSectionListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
}

type ThreadSectionCreateRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params ThreadSectionCreateParams `json:"params"`
}

type ThreadSectionCreateParams struct {
	Name string `json:"name"`
}

type ThreadSectionUpdateRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params ThreadSectionUpdateParams `json:"params"`
}

type ThreadSectionUpdateParams struct {
	Name      string `json:"name"`
	SectionID string `json:"sectionId"`
}

type ThreadSectionDeleteRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params ThreadSectionDeleteParams `json:"params"`
}

type ThreadSectionDeleteParams struct {
	SectionID string `json:"sectionId"`
}

type ThreadLoadedListRequestEnvelope struct {
	ID     RequestId              `json:"id"`
	Method string                 `json:"method"`
	Params ThreadLoadedListParams `json:"params"`
}

type ThreadLoadedListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
}

type ThreadReadRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params ThreadReadParams `json:"params"`
}

type ThreadReadParams struct {
	IncludeTurns *bool  `json:"includeTurns,omitempty"`
	ThreadID     string `json:"threadId"`
}

type ThreadInjectItemsRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params ThreadInjectItemsParams `json:"params"`
}

type ThreadInjectItemsParams struct {
	Items    []any  `json:"items"`
	ThreadID string `json:"threadId"`
}

type SkillsListRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params SkillsListParams `json:"params"`
}

type SkillsListParams struct {
	Cwds        []string `json:"cwds,omitempty"`
	ForceReload *bool    `json:"forceReload,omitempty"`
}

type SkillsExtraRootsSetRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params SkillsExtraRootsSetParams `json:"params"`
}

type SkillsExtraRootsSetParams struct {
	ExtraRoots []AbsolutePathBuf `json:"extraRoots"`
}

type HooksListRequestEnvelope struct {
	ID     RequestId       `json:"id"`
	Method string          `json:"method"`
	Params HooksListParams `json:"params"`
}

type HooksListParams struct {
	Cwds []string `json:"cwds,omitempty"`
}

type MarketplaceAddRequestEnvelope struct {
	ID     RequestId            `json:"id"`
	Method string               `json:"method"`
	Params MarketplaceAddParams `json:"params"`
}

type MarketplaceAddParams struct {
	RefName     *string  `json:"refName,omitempty"`
	Source      string   `json:"source"`
	SparsePaths []string `json:"sparsePaths,omitempty"`
}

type MarketplaceRemoveRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params MarketplaceRemoveParams `json:"params"`
}

type MarketplaceRemoveParams struct {
	MarketplaceName string `json:"marketplaceName"`
}

type MarketplaceUpgradeRequestEnvelope struct {
	ID     RequestId                `json:"id"`
	Method string                   `json:"method"`
	Params MarketplaceUpgradeParams `json:"params"`
}

type MarketplaceUpgradeParams struct {
	MarketplaceName *string `json:"marketplaceName,omitempty"`
}

type PluginListRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params PluginListParams `json:"params"`
}

type PluginListParams struct {
	Cwds             []AbsolutePathBuf           `json:"cwds,omitempty"`
	ForceRefetch     *bool                       `json:"forceRefetch,omitempty"`
	MarketplaceKinds []PluginListMarketplaceKind `json:"marketplaceKinds,omitempty"`
}

type PluginListMarketplaceKind string

const (
	PluginListMarketplaceKindLocal              PluginListMarketplaceKind = "local"
	PluginListMarketplaceKindVertical           PluginListMarketplaceKind = "vertical"
	PluginListMarketplaceKindWorkspaceDirectory PluginListMarketplaceKind = "workspace-directory"
	PluginListMarketplaceKindSharedWithMe       PluginListMarketplaceKind = "shared-with-me"
	PluginListMarketplaceKindCreatedByMeRemote  PluginListMarketplaceKind = "created-by-me-remote"
)

type PluginInstalledRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params PluginInstalledParams `json:"params"`
}

type PluginInstalledParams struct {
	Cwds                         []AbsolutePathBuf `json:"cwds,omitempty"`
	InstallSuggestionPluginNames []string          `json:"installSuggestionPluginNames,omitempty"`
}

type PluginReadRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params PluginReadParams `json:"params"`
}

type PluginReadParams struct {
	MarketplacePath       *AbsolutePathBuf `json:"marketplacePath,omitempty"`
	PluginName            string           `json:"pluginName"`
	RemoteMarketplaceName *string          `json:"remoteMarketplaceName,omitempty"`
}

type PluginSkillReadRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params PluginSkillReadParams `json:"params"`
}

type PluginSkillReadParams struct {
	RemoteMarketplaceName string `json:"remoteMarketplaceName"`
	RemotePluginID        string `json:"remotePluginId"`
	SkillName             string `json:"skillName"`
}

type PluginShareSaveRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params PluginShareSaveParams `json:"params"`
}

type PluginShareSaveParams struct {
	Discoverability *PluginShareDiscoverability `json:"discoverability,omitempty"`
	PluginPath      AbsolutePathBuf             `json:"pluginPath"`
	RemotePluginID  *string                     `json:"remotePluginId,omitempty"`
	ShareTargets    []PluginShareTarget         `json:"shareTargets,omitempty"`
}

type PluginShareDiscoverability string

const (
	PluginShareDiscoverabilityLISTED   PluginShareDiscoverability = "LISTED"
	PluginShareDiscoverabilityUNLISTED PluginShareDiscoverability = "UNLISTED"
	PluginShareDiscoverabilityPRIVATE  PluginShareDiscoverability = "PRIVATE"
)

type PluginShareTarget struct {
	PrincipalID   string                   `json:"principalId"`
	PrincipalType PluginSharePrincipalType `json:"principalType"`
	Role          PluginShareTargetRole    `json:"role"`
}

type PluginSharePrincipalType string

const (
	PluginSharePrincipalTypeUser      PluginSharePrincipalType = "user"
	PluginSharePrincipalTypeGroup     PluginSharePrincipalType = "group"
	PluginSharePrincipalTypeWorkspace PluginSharePrincipalType = "workspace"
)

type PluginShareTargetRole string

const (
	PluginShareTargetRoleReader PluginShareTargetRole = "reader"
	PluginShareTargetRoleEditor PluginShareTargetRole = "editor"
)

type PluginShareUpdateTargetsRequestEnvelope struct {
	ID     RequestId                      `json:"id"`
	Method string                         `json:"method"`
	Params PluginShareUpdateTargetsParams `json:"params"`
}

type PluginShareUpdateTargetsParams struct {
	Discoverability PluginShareUpdateDiscoverability `json:"discoverability"`
	RemotePluginID  string                           `json:"remotePluginId"`
	ShareTargets    []PluginShareTarget              `json:"shareTargets"`
}

type PluginShareUpdateDiscoverability string

const (
	PluginShareUpdateDiscoverabilityUNLISTED PluginShareUpdateDiscoverability = "UNLISTED"
	PluginShareUpdateDiscoverabilityPRIVATE  PluginShareUpdateDiscoverability = "PRIVATE"
	PluginShareUpdateDiscoverabilityLISTED   PluginShareUpdateDiscoverability = "LISTED"
)

type PluginShareListRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params PluginShareListParams `json:"params"`
}

type PluginShareListParams struct {
}

type PluginShareCheckoutRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params PluginShareCheckoutParams `json:"params"`
}

type PluginShareCheckoutParams struct {
	RemotePluginID string `json:"remotePluginId"`
}

type PluginShareDeleteRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params PluginShareDeleteParams `json:"params"`
}

type PluginShareDeleteParams struct {
	RemotePluginID string `json:"remotePluginId"`
}

type AppReadRequestEnvelope struct {
	ID     RequestId      `json:"id"`
	Method string         `json:"method"`
	Params AppsReadParams `json:"params"`
}

type AppListRequestEnvelope struct {
	ID     RequestId      `json:"id"`
	Method string         `json:"method"`
	Params AppsListParams `json:"params"`
}

type AppInstalledRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params AppsInstalledParams `json:"params"`
}

type FsReadFileRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params FsReadFileParams `json:"params"`
}

type FsReadFileParams struct {
	Path AbsolutePathBuf `json:"path"`
}

type FsWriteFileRequestEnvelope struct {
	ID     RequestId         `json:"id"`
	Method string            `json:"method"`
	Params FsWriteFileParams `json:"params"`
}

type FsWriteFileParams struct {
	DataBase64 string          `json:"dataBase64"`
	Path       AbsolutePathBuf `json:"path"`
}

type FsCreateDirectoryRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params FsCreateDirectoryParams `json:"params"`
}

type FsCreateDirectoryParams struct {
	Path      AbsolutePathBuf `json:"path"`
	Recursive *bool           `json:"recursive,omitempty"`
}

type FsGetMetadataRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params FsGetMetadataParams `json:"params"`
}

type FsGetMetadataParams struct {
	Path AbsolutePathBuf `json:"path"`
}

type FsReadDirectoryRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params FsReadDirectoryParams `json:"params"`
}

type FsReadDirectoryParams struct {
	Path AbsolutePathBuf `json:"path"`
}

type FsRemoveRequestEnvelope struct {
	ID     RequestId      `json:"id"`
	Method string         `json:"method"`
	Params FsRemoveParams `json:"params"`
}

type FsRemoveParams struct {
	Force     *bool           `json:"force,omitempty"`
	Path      AbsolutePathBuf `json:"path"`
	Recursive *bool           `json:"recursive,omitempty"`
}

type FsCopyRequestEnvelope struct {
	ID     RequestId    `json:"id"`
	Method string       `json:"method"`
	Params FsCopyParams `json:"params"`
}

type FsCopyParams struct {
	DestinationPath AbsolutePathBuf `json:"destinationPath"`
	Recursive       *bool           `json:"recursive,omitempty"`
	SourcePath      AbsolutePathBuf `json:"sourcePath"`
}

type FsWatchRequestEnvelope struct {
	ID     RequestId     `json:"id"`
	Method string        `json:"method"`
	Params FsWatchParams `json:"params"`
}

type FsWatchParams struct {
	Path    AbsolutePathBuf `json:"path"`
	WatchID string          `json:"watchId"`
}

type FsUnwatchRequestEnvelope struct {
	ID     RequestId       `json:"id"`
	Method string          `json:"method"`
	Params FsUnwatchParams `json:"params"`
}

type FsUnwatchParams struct {
	WatchID string `json:"watchId"`
}

type SkillsConfigWriteRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params SkillsConfigWriteParams `json:"params"`
}

type SkillsConfigWriteParams struct {
	Enabled bool             `json:"enabled"`
	Name    *string          `json:"name,omitempty"`
	Path    *AbsolutePathBuf `json:"path,omitempty"`
}

type PluginInstallRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params PluginInstallParams `json:"params"`
}

type PluginInstallParams struct {
	MarketplacePath       *AbsolutePathBuf `json:"marketplacePath,omitempty"`
	PluginName            string           `json:"pluginName"`
	RemoteMarketplaceName *string          `json:"remoteMarketplaceName,omitempty"`
}

type PluginUninstallRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params PluginUninstallParams `json:"params"`
}

type PluginUninstallParams struct {
	PluginID string `json:"pluginId"`
}

type TurnStartRequestEnvelope struct {
	ID     RequestId       `json:"id"`
	Method string          `json:"method"`
	Params TurnStartParams `json:"params"`
}

type TurnStartParams struct {
	ApprovalPolicy      *AskForApproval    `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer   *ApprovalsReviewer `json:"approvalsReviewer,omitempty"`
	ClientUserMessageID *string            `json:"clientUserMessageId,omitempty"`
	Cwd                 *string            `json:"cwd,omitempty"`
	Effort              *ReasoningEffort   `json:"effort,omitempty"`
	Input               []UserInput        `json:"input"`
	Model               *string            `json:"model,omitempty"`
	OutputSchema        any                `json:"outputSchema,omitempty"`
	Personality         *Personality       `json:"personality,omitempty"`
	SandboxPolicy       *SandboxPolicy     `json:"sandboxPolicy,omitempty"`
	ServiceTier         *string            `json:"serviceTier,omitempty"`
	Summary             *ReasoningSummary  `json:"summary,omitempty"`
	ThreadID            string             `json:"threadId"`
}

type ReasoningEffort string

type UserInput struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *UserInput) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v TextUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "image":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type", "url")) {
					break
				}
				var v ImageUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "localImage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "path", "type")) {
					break
				}
				var v LocalImageUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "audio":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type", "url")) {
					break
				}
				var v AudioUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "localAudio":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "path", "type")) {
					break
				}
				var v LocalAudioUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "skill":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "name", "path", "type")) {
					break
				}
				var v SkillUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mention":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "name", "path", "type")) {
					break
				}
				var v MentionUserInput
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v TextUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type", "url") {
		var v ImageUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "path", "type") {
		var v LocalImageUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type", "url") {
		var v AudioUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "path", "type") {
		var v LocalAudioUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "name", "path", "type") {
		var v SkillUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "name", "path", "type") {
		var v MentionUserInput
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u UserInput) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type TextUserInput struct {
	Text         string        `json:"text"`
	TextElements []TextElement `json:"text_elements,omitempty"`
	Type         string        `json:"type"`
}

type TextElement struct {
	ByteRange   ByteRange `json:"byteRange"`
	Placeholder *string   `json:"placeholder,omitempty"`
}

type ImageUserInput struct {
	Detail *ImageDetail `json:"detail,omitempty"`
	Type   string       `json:"type"`
	Url    string       `json:"url"`
}

type ImageDetail string

const (
	ImageDetailAuto     ImageDetail = "auto"
	ImageDetailLow      ImageDetail = "low"
	ImageDetailHigh     ImageDetail = "high"
	ImageDetailOriginal ImageDetail = "original"
)

type LocalImageUserInput struct {
	Detail *ImageDetail `json:"detail,omitempty"`
	Path   string       `json:"path"`
	Type   string       `json:"type"`
}

type AudioUserInput struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

type LocalAudioUserInput struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type SkillUserInput struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type MentionUserInput struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type SandboxPolicy struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *SandboxPolicy) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "dangerFullAccess":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v DangerFullAccessSandboxPolicy
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "readOnly":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ReadOnlySandboxPolicy
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalSandbox":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ExternalSandboxSandboxPolicy
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "workspaceWrite":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v WorkspaceWriteSandboxPolicy
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v DangerFullAccessSandboxPolicy
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ReadOnlySandboxPolicy
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ExternalSandboxSandboxPolicy
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v WorkspaceWriteSandboxPolicy
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u SandboxPolicy) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type DangerFullAccessSandboxPolicy struct {
	Type string `json:"type"`
}

type ReadOnlySandboxPolicy struct {
	NetworkAccess *bool  `json:"networkAccess,omitempty"`
	Type          string `json:"type"`
}

type ExternalSandboxSandboxPolicy struct {
	NetworkAccess *NetworkAccess `json:"networkAccess,omitempty"`
	Type          string         `json:"type"`
}

type NetworkAccess string

const (
	NetworkAccessRestricted NetworkAccess = "restricted"
	NetworkAccessEnabled    NetworkAccess = "enabled"
)

type WorkspaceWriteSandboxPolicy struct {
	ExcludeSlashTmp     *bool             `json:"excludeSlashTmp,omitempty"`
	ExcludeTmpdirEnvVar *bool             `json:"excludeTmpdirEnvVar,omitempty"`
	NetworkAccess       *bool             `json:"networkAccess,omitempty"`
	Type                string            `json:"type"`
	WritableRoots       []AbsolutePathBuf `json:"writableRoots,omitempty"`
}

type ReasoningSummary struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ReasoningSummary) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "auto", "concise", "detailed") {
		var v ReasoningSummaryVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "none") {
		var v ReasoningSummaryVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ReasoningSummary) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ReasoningSummaryVariant1 string

const (
	ReasoningSummaryVariant1Auto     ReasoningSummaryVariant1 = "auto"
	ReasoningSummaryVariant1Concise  ReasoningSummaryVariant1 = "concise"
	ReasoningSummaryVariant1Detailed ReasoningSummaryVariant1 = "detailed"
)

type ReasoningSummaryVariant2 string

const (
	ReasoningSummaryVariant2None ReasoningSummaryVariant2 = "none"
)

type TurnSteerRequestEnvelope struct {
	ID     RequestId       `json:"id"`
	Method string          `json:"method"`
	Params TurnSteerParams `json:"params"`
}

type TurnSteerParams struct {
	ClientUserMessageID *string     `json:"clientUserMessageId,omitempty"`
	ExpectedTurnID      string      `json:"expectedTurnId"`
	Input               []UserInput `json:"input"`
	ThreadID            string      `json:"threadId"`
}

type TurnInterruptRequestEnvelope struct {
	ID     RequestId           `json:"id"`
	Method string              `json:"method"`
	Params TurnInterruptParams `json:"params"`
}

type TurnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type ReviewStartRequestEnvelope struct {
	ID     RequestId         `json:"id"`
	Method string            `json:"method"`
	Params ReviewStartParams `json:"params"`
}

type ReviewStartParams struct {
	Delivery *ReviewDelivery `json:"delivery,omitempty"`
	Target   ReviewTarget    `json:"target"`
	ThreadID string          `json:"threadId"`
}

type ReviewDelivery string

const (
	ReviewDeliveryInline   ReviewDelivery = "inline"
	ReviewDeliveryDetached ReviewDelivery = "detached"
)

type ReviewTarget struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ReviewTarget) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "uncommittedChanges":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v UncommittedChangesReviewTarget
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "baseBranch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "branch", "type")) {
					break
				}
				var v BaseBranchReviewTarget
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "commit":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "sha", "type")) {
					break
				}
				var v CommitReviewTarget
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "custom":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "instructions", "type")) {
					break
				}
				var v CustomReviewTarget
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v UncommittedChangesReviewTarget
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "branch", "type") {
		var v BaseBranchReviewTarget
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "sha", "type") {
		var v CommitReviewTarget
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "instructions", "type") {
		var v CustomReviewTarget
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ReviewTarget) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type UncommittedChangesReviewTarget struct {
	Type string `json:"type"`
}

type BaseBranchReviewTarget struct {
	Branch string `json:"branch"`
	Type   string `json:"type"`
}

type CommitReviewTarget struct {
	Sha   string  `json:"sha"`
	Title *string `json:"title,omitempty"`
	Type  string  `json:"type"`
}

type CustomReviewTarget struct {
	Instructions string `json:"instructions"`
	Type         string `json:"type"`
}

type ModelListRequestEnvelope struct {
	ID     RequestId       `json:"id"`
	Method string          `json:"method"`
	Params ModelListParams `json:"params"`
}

type ModelListParams struct {
	Cursor        *string `json:"cursor,omitempty"`
	IncludeHidden *bool   `json:"includeHidden,omitempty"`
	Limit         *int64  `json:"limit,omitempty"`
}

type ModelProviderCapabilitiesReadRequestEnvelope struct {
	ID     RequestId                           `json:"id"`
	Method string                              `json:"method"`
	Params ModelProviderCapabilitiesReadParams `json:"params"`
}

type ModelProviderCapabilitiesReadParams struct {
}

type ExperimentalFeatureListRequestEnvelope struct {
	ID     RequestId                     `json:"id"`
	Method string                        `json:"method"`
	Params ExperimentalFeatureListParams `json:"params"`
}

type ExperimentalFeatureListParams struct {
	Cursor   *string `json:"cursor,omitempty"`
	Limit    *int64  `json:"limit,omitempty"`
	ThreadID *string `json:"threadId,omitempty"`
}

type PermissionProfileListRequestEnvelope struct {
	ID     RequestId                   `json:"id"`
	Method string                      `json:"method"`
	Params PermissionProfileListParams `json:"params"`
}

type PermissionProfileListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Cwd    *string `json:"cwd,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
}

type ExperimentalFeatureEnablementSetRequestEnvelope struct {
	ID     RequestId                              `json:"id"`
	Method string                                 `json:"method"`
	Params ExperimentalFeatureEnablementSetParams `json:"params"`
}

type ExperimentalFeatureEnablementSetParams struct {
	Enablement map[string]bool `json:"enablement"`
}

type McpServerOauthLoginRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params McpServerOauthLoginParams `json:"params"`
}

type McpServerOauthLoginParams struct {
	Name        string   `json:"name"`
	Scopes      []string `json:"scopes,omitempty"`
	ThreadID    *string  `json:"threadId,omitempty"`
	TimeoutSecs *int64   `json:"timeoutSecs,omitempty"`
}

type ConfigMcpServerReloadRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type McpServerStatusListRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params ListMcpServerStatusParams `json:"params"`
}

type ListMcpServerStatusParams struct {
	Cursor   *string                `json:"cursor,omitempty"`
	Detail   *McpServerStatusDetail `json:"detail,omitempty"`
	Limit    *int64                 `json:"limit,omitempty"`
	ThreadID *string                `json:"threadId,omitempty"`
}

type McpServerStatusDetail string

const (
	McpServerStatusDetailFull             McpServerStatusDetail = "full"
	McpServerStatusDetailToolsAndAuthOnly McpServerStatusDetail = "toolsAndAuthOnly"
)

type McpServerResourceReadRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params McpResourceReadParams `json:"params"`
}

type McpResourceReadParams struct {
	Server   string  `json:"server"`
	ThreadID *string `json:"threadId,omitempty"`
	Uri      string  `json:"uri"`
}

type McpServerToolCallRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params McpServerToolCallParams `json:"params"`
}

type McpServerToolCallParams struct {
	Meta      any    `json:"_meta,omitempty"`
	Arguments any    `json:"arguments,omitempty"`
	Server    string `json:"server"`
	ThreadID  string `json:"threadId"`
	Tool      string `json:"tool"`
}

type WindowsSandboxSetupStartRequestEnvelope struct {
	ID     RequestId                      `json:"id"`
	Method string                         `json:"method"`
	Params WindowsSandboxSetupStartParams `json:"params"`
}

type WindowsSandboxSetupStartParams struct {
	Cwd  *AbsolutePathBuf        `json:"cwd,omitempty"`
	Mode WindowsSandboxSetupMode `json:"mode"`
}

type WindowsSandboxSetupMode string

const (
	WindowsSandboxSetupModeElevated   WindowsSandboxSetupMode = "elevated"
	WindowsSandboxSetupModeUnelevated WindowsSandboxSetupMode = "unelevated"
)

type WindowsSandboxReadinessRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountLoginStartRequestEnvelope struct {
	ID     RequestId          `json:"id"`
	Method string             `json:"method"`
	Params LoginAccountParams `json:"params"`
}

type LoginAccountParams struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *LoginAccountParams) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "apiKey":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "apiKey", "type")) {
					break
				}
				var v ApiKeyv2LoginAccountParams
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgpt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v Chatgptv2LoginAccountParams
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgptDeviceCode":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ChatgptDeviceCodev2LoginAccountParams
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgptAuthTokens":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "accessToken", "chatgptAccountId", "type")) {
					break
				}
				var v ChatgptAuthTokensv2LoginAccountParams
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "amazonBedrock":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "apiKey", "region", "type")) {
					break
				}
				var v AmazonBedrockv2LoginAccountParams
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "apiKey", "type") {
		var v ApiKeyv2LoginAccountParams
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v Chatgptv2LoginAccountParams
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ChatgptDeviceCodev2LoginAccountParams
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "accessToken", "chatgptAccountId", "type") {
		var v ChatgptAuthTokensv2LoginAccountParams
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "apiKey", "region", "type") {
		var v AmazonBedrockv2LoginAccountParams
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u LoginAccountParams) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ApiKeyv2LoginAccountParams struct {
	ApiKey string `json:"apiKey"`
	Type   string `json:"type"`
}

type Chatgptv2LoginAccountParams struct {
	AppBrand                  *LoginAppBrand `json:"appBrand,omitempty"`
	CodexStreamlinedLogin     *bool          `json:"codexStreamlinedLogin,omitempty"`
	Type                      string         `json:"type"`
	UseHostedLoginSuccessPage *bool          `json:"useHostedLoginSuccessPage,omitempty"`
}

type LoginAppBrand string

const (
	LoginAppBrandCodex   LoginAppBrand = "codex"
	LoginAppBrandChatgpt LoginAppBrand = "chatgpt"
)

type ChatgptDeviceCodev2LoginAccountParams struct {
	Type string `json:"type"`
}

type ChatgptAuthTokensv2LoginAccountParams struct {
	AccessToken      string  `json:"accessToken"`
	ChatgptAccountID string  `json:"chatgptAccountId"`
	ChatgptPlanType  *string `json:"chatgptPlanType,omitempty"`
	Type             string  `json:"type"`
}

type AmazonBedrockv2LoginAccountParams struct {
	ApiKey string `json:"apiKey"`
	Region string `json:"region"`
	Type   string `json:"type"`
}

type AccountLoginCancelRequestEnvelope struct {
	ID     RequestId                `json:"id"`
	Method string                   `json:"method"`
	Params CancelLoginAccountParams `json:"params"`
}

type AccountLogoutRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountRateLimitsReadRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountRateLimitResetCreditConsumeRequestEnvelope struct {
	ID     RequestId                                `json:"id"`
	Method string                                   `json:"method"`
	Params ConsumeAccountRateLimitResetCreditParams `json:"params"`
}

type ConsumeAccountRateLimitResetCreditParams struct {
	CreditID       *string `json:"creditId,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey"`
}

type AccountUsageReadRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountWorkspaceMessagesReadRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountSendAddCreditsNudgeEmailRequestEnvelope struct {
	ID     RequestId                      `json:"id"`
	Method string                         `json:"method"`
	Params SendAddCreditsNudgeEmailParams `json:"params"`
}

type SendAddCreditsNudgeEmailParams struct {
	CreditType AddCreditsNudgeCreditType `json:"creditType"`
}

type FeedbackUploadRequestEnvelope struct {
	ID     RequestId            `json:"id"`
	Method string               `json:"method"`
	Params FeedbackUploadParams `json:"params"`
}

type FeedbackUploadParams struct {
	Classification string            `json:"classification"`
	ExtraLogFiles  []string          `json:"extraLogFiles,omitempty"`
	IncludeLogs    *bool             `json:"includeLogs,omitempty"`
	Reason         *string           `json:"reason,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	ThreadID       *string           `json:"threadId,omitempty"`
}

type CommandExecRequestEnvelope struct {
	ID     RequestId         `json:"id"`
	Method string            `json:"method"`
	Params CommandExecParams `json:"params"`
}

type CommandExecParams struct {
	Command            []string                 `json:"command"`
	Cwd                *string                  `json:"cwd,omitempty"`
	DisableOutputCap   *bool                    `json:"disableOutputCap,omitempty"`
	DisableTimeout     *bool                    `json:"disableTimeout,omitempty"`
	Env                map[string]string        `json:"env,omitempty"`
	OutputBytesCap     *int64                   `json:"outputBytesCap,omitempty"`
	ProcessID          *string                  `json:"processId,omitempty"`
	SandboxPolicy      *SandboxPolicy           `json:"sandboxPolicy,omitempty"`
	Size               *CommandExecTerminalSize `json:"size,omitempty"`
	StreamStdin        *bool                    `json:"streamStdin,omitempty"`
	StreamStdoutStderr *bool                    `json:"streamStdoutStderr,omitempty"`
	TimeoutMs          *int64                   `json:"timeoutMs,omitempty"`
	Tty                *bool                    `json:"tty,omitempty"`
}

type CommandExecTerminalSize struct {
	Cols int64 `json:"cols"`
	Rows int64 `json:"rows"`
}

type CommandExecWriteRequestEnvelope struct {
	ID     RequestId              `json:"id"`
	Method string                 `json:"method"`
	Params CommandExecWriteParams `json:"params"`
}

type CommandExecWriteParams struct {
	CloseStdin  *bool   `json:"closeStdin,omitempty"`
	DeltaBase64 *string `json:"deltaBase64,omitempty"`
	ProcessID   string  `json:"processId"`
}

type CommandExecTerminateRequestEnvelope struct {
	ID     RequestId                  `json:"id"`
	Method string                     `json:"method"`
	Params CommandExecTerminateParams `json:"params"`
}

type CommandExecTerminateParams struct {
	ProcessID string `json:"processId"`
}

type CommandExecResizeRequestEnvelope struct {
	ID     RequestId               `json:"id"`
	Method string                  `json:"method"`
	Params CommandExecResizeParams `json:"params"`
}

type CommandExecResizeParams struct {
	ProcessID string                  `json:"processId"`
	Size      CommandExecTerminalSize `json:"size"`
}

type ConfigReadRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params ConfigReadParams `json:"params"`
}

type ConfigReadParams struct {
	Cwd           *string `json:"cwd,omitempty"`
	IncludeLayers *bool   `json:"includeLayers,omitempty"`
}

type ExternalAgentConfigDetectRequestEnvelope struct {
	ID     RequestId                       `json:"id"`
	Method string                          `json:"method"`
	Params ExternalAgentConfigDetectParams `json:"params"`
}

type ExternalAgentConfigDetectParams struct {
	Cwds              []string `json:"cwds,omitempty"`
	IncludeHome       *bool    `json:"includeHome,omitempty"`
	MaxSessionAgeDays *int64   `json:"maxSessionAgeDays,omitempty"`
	MaxSessions       *int64   `json:"maxSessions,omitempty"`
	MigrationSource   *string  `json:"migrationSource,omitempty"`
	Source            *string  `json:"source,omitempty"`
}

type ExternalAgentConfigImportRequestEnvelope struct {
	ID     RequestId                       `json:"id"`
	Method string                          `json:"method"`
	Params ExternalAgentConfigImportParams `json:"params"`
}

type ExternalAgentConfigImportParams struct {
	MigrationItems  []ExternalAgentConfigMigrationItem `json:"migrationItems"`
	MigrationSource *string                            `json:"migrationSource,omitempty"`
	ProviderID      *string                            `json:"providerId,omitempty"`
	Source          *string                            `json:"source,omitempty"`
}

type ExternalAgentConfigMigrationItem struct {
	Cwd         *string                              `json:"cwd,omitempty"`
	Description string                               `json:"description"`
	Details     *MigrationDetails                    `json:"details,omitempty"`
	ItemType    ExternalAgentConfigMigrationItemType `json:"itemType"`
}

type MigrationDetails struct {
	Commands   []CommandMigration   `json:"commands,omitempty"`
	Hooks      []HookMigration      `json:"hooks,omitempty"`
	McpServers []McpServerMigration `json:"mcpServers,omitempty"`
	Memory     []string             `json:"memory,omitempty"`
	Plugins    []PluginsMigration   `json:"plugins,omitempty"`
	Sessions   []SessionMigration   `json:"sessions,omitempty"`
	Skills     []SkillMigration     `json:"skills,omitempty"`
	Subagents  []SubagentMigration  `json:"subagents,omitempty"`
}

type CommandMigration struct {
	Name string `json:"name"`
}

type HookMigration struct {
	Name string `json:"name"`
}

type McpServerMigration struct {
	Name string `json:"name"`
}

type PluginsMigration struct {
	MarketplaceName string   `json:"marketplaceName"`
	PluginNames     []string `json:"pluginNames"`
}

type SessionMigration struct {
	Cwd   string  `json:"cwd"`
	Path  string  `json:"path"`
	Title *string `json:"title,omitempty"`
}

type SkillMigration struct {
	Name string `json:"name"`
}

type SubagentMigration struct {
	Name string `json:"name"`
}

type ExternalAgentConfigMigrationItemType string

const (
	ExternalAgentConfigMigrationItemTypeAGENTSMD        ExternalAgentConfigMigrationItemType = "AGENTS_MD"
	ExternalAgentConfigMigrationItemTypeCONFIG          ExternalAgentConfigMigrationItemType = "CONFIG"
	ExternalAgentConfigMigrationItemTypeSKILLS          ExternalAgentConfigMigrationItemType = "SKILLS"
	ExternalAgentConfigMigrationItemTypePLUGINS         ExternalAgentConfigMigrationItemType = "PLUGINS"
	ExternalAgentConfigMigrationItemTypeMCPSERVERCONFIG ExternalAgentConfigMigrationItemType = "MCP_SERVER_CONFIG"
	ExternalAgentConfigMigrationItemTypeSUBAGENTS       ExternalAgentConfigMigrationItemType = "SUBAGENTS"
	ExternalAgentConfigMigrationItemTypeHOOKS           ExternalAgentConfigMigrationItemType = "HOOKS"
	ExternalAgentConfigMigrationItemTypeCOMMANDS        ExternalAgentConfigMigrationItemType = "COMMANDS"
	ExternalAgentConfigMigrationItemTypeMEMORY          ExternalAgentConfigMigrationItemType = "MEMORY"
	ExternalAgentConfigMigrationItemTypeSESSIONS        ExternalAgentConfigMigrationItemType = "SESSIONS"
)

type ExternalAgentConfigImportRecordHistoryRequestEnvelope struct {
	ID     RequestId                                    `json:"id"`
	Method string                                       `json:"method"`
	Params ExternalAgentConfigImportHistoryRecordParams `json:"params"`
}

type ExternalAgentConfigImportHistoryRecordParams struct {
	ItemTypeResults []ExternalAgentConfigImportHistoryRecordTypeResultParams `json:"itemTypeResults"`
	ProviderID      string                                                   `json:"providerId"`
}

type ExternalAgentConfigImportHistoryRecordTypeResultParams struct {
	Failures  []ExternalAgentConfigImportItemTypeFailure            `json:"failures"`
	ItemType  ExternalAgentConfigMigrationItemType                  `json:"itemType"`
	Successes []ExternalAgentConfigImportHistoryRecordSuccessParams `json:"successes"`
}

type ExternalAgentConfigImportItemTypeFailure struct {
	Cwd          *string                              `json:"cwd,omitempty"`
	ErrorType    *string                              `json:"errorType,omitempty"`
	FailureStage string                               `json:"failureStage"`
	ItemType     ExternalAgentConfigMigrationItemType `json:"itemType"`
	Message      string                               `json:"message"`
	Source       *string                              `json:"source,omitempty"`
	SubErrorType *string                              `json:"subErrorType,omitempty"`
}

type ExternalAgentConfigImportHistoryRecordSuccessParams struct {
	Cwd      *string                              `json:"cwd,omitempty"`
	ItemType ExternalAgentConfigMigrationItemType `json:"itemType"`
	Source   *string                              `json:"source,omitempty"`
	Target   *string                              `json:"target,omitempty"`
	Title    *string                              `json:"title,omitempty"`
}

type ExternalAgentConfigImportReadHistoriesRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type ConfigValueWriteRequestEnvelope struct {
	ID     RequestId              `json:"id"`
	Method string                 `json:"method"`
	Params ConfigValueWriteParams `json:"params"`
}

type ConfigValueWriteParams struct {
	ExpectedVersion *string       `json:"expectedVersion,omitempty"`
	FilePath        *string       `json:"filePath,omitempty"`
	KeyPath         string        `json:"keyPath"`
	MergeStrategy   MergeStrategy `json:"mergeStrategy"`
	Value           any           `json:"value"`
}

type MergeStrategy string

const (
	MergeStrategyReplace MergeStrategy = "replace"
	MergeStrategyUpsert  MergeStrategy = "upsert"
)

type ConfigBatchWriteRequestEnvelope struct {
	ID     RequestId              `json:"id"`
	Method string                 `json:"method"`
	Params ConfigBatchWriteParams `json:"params"`
}

type ConfigBatchWriteParams struct {
	Edits            []ConfigEdit `json:"edits"`
	ExpectedVersion  *string      `json:"expectedVersion,omitempty"`
	FilePath         *string      `json:"filePath,omitempty"`
	ReloadUserConfig *bool        `json:"reloadUserConfig,omitempty"`
}

type ConfigEdit struct {
	KeyPath       string        `json:"keyPath"`
	MergeStrategy MergeStrategy `json:"mergeStrategy"`
	Value         any           `json:"value"`
}

type ConfigRequirementsReadRequestEnvelope struct {
	ID     RequestId `json:"id"`
	Method string    `json:"method"`
	Params any       `json:"params,omitempty"`
}

type AccountReadRequestEnvelope struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params GetAccountParams `json:"params"`
}

type GetAccountParams struct {
	RefreshToken *bool `json:"refreshToken,omitempty"`
}

type FuzzyFileSearchRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params FuzzyFileSearchParams `json:"params"`
}

type FuzzyFileSearchParams struct {
	CancellationToken *string  `json:"cancellationToken,omitempty"`
	Query             string   `json:"query"`
	Roots             []string `json:"roots"`
}

type CodexErrorInfo struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *CodexErrorInfo) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "contextWindowExceeded", "sessionBudgetExceeded", "usageLimitExceeded", "serverOverloaded", "cyberPolicy", "internalServerError", "unauthorized", "badRequest", "threadRollbackFailed", "sandboxError", "other") {
		var v CodexErrorInfoVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "httpConnectionFailed") {
		var v HttpConnectionFailedCodexErrorInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "responseStreamConnectionFailed") {
		var v ResponseStreamConnectionFailedCodexErrorInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "responseStreamDisconnected") {
		var v ResponseStreamDisconnectedCodexErrorInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "responseTooManyFailedAttempts") {
		var v ResponseTooManyFailedAttemptsCodexErrorInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "activeTurnNotSteerable") {
		var v ActiveTurnNotSteerableCodexErrorInfo
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u CodexErrorInfo) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type CodexErrorInfoVariant1 string

const (
	CodexErrorInfoVariant1ContextWindowExceeded CodexErrorInfoVariant1 = "contextWindowExceeded"
	CodexErrorInfoVariant1SessionBudgetExceeded CodexErrorInfoVariant1 = "sessionBudgetExceeded"
	CodexErrorInfoVariant1UsageLimitExceeded    CodexErrorInfoVariant1 = "usageLimitExceeded"
	CodexErrorInfoVariant1ServerOverloaded      CodexErrorInfoVariant1 = "serverOverloaded"
	CodexErrorInfoVariant1CyberPolicy           CodexErrorInfoVariant1 = "cyberPolicy"
	CodexErrorInfoVariant1InternalServerError   CodexErrorInfoVariant1 = "internalServerError"
	CodexErrorInfoVariant1Unauthorized          CodexErrorInfoVariant1 = "unauthorized"
	CodexErrorInfoVariant1BadRequest            CodexErrorInfoVariant1 = "badRequest"
	CodexErrorInfoVariant1ThreadRollbackFailed  CodexErrorInfoVariant1 = "threadRollbackFailed"
	CodexErrorInfoVariant1SandboxError          CodexErrorInfoVariant1 = "sandboxError"
	CodexErrorInfoVariant1Other                 CodexErrorInfoVariant1 = "other"
)

type HttpConnectionFailedCodexErrorInfo struct {
	HttpConnectionFailed map[string]any `json:"httpConnectionFailed"`
}

type ResponseStreamConnectionFailedCodexErrorInfo struct {
	ResponseStreamConnectionFailed map[string]any `json:"responseStreamConnectionFailed"`
}

type ResponseStreamDisconnectedCodexErrorInfo struct {
	ResponseStreamDisconnected map[string]any `json:"responseStreamDisconnected"`
}

type ResponseTooManyFailedAttemptsCodexErrorInfo struct {
	ResponseTooManyFailedAttempts map[string]any `json:"responseTooManyFailedAttempts"`
}

type ActiveTurnNotSteerableCodexErrorInfo struct {
	ActiveTurnNotSteerable map[string]any `json:"activeTurnNotSteerable"`
}

type CodexResponseHandoffMode string

const (
	CodexResponseHandoffModeThinking   CodexResponseHandoffMode = "thinking"
	CodexResponseHandoffModeCommentary CodexResponseHandoffMode = "commentary"
	CodexResponseHandoffModeBemTags    CodexResponseHandoffMode = "bemTags"
)

type CollabAgentState struct {
	Message *string           `json:"message,omitempty"`
	Status  CollabAgentStatus `json:"status"`
}

type CollabAgentStatus string

const (
	CollabAgentStatusPendingInit CollabAgentStatus = "pendingInit"
	CollabAgentStatusRunning     CollabAgentStatus = "running"
	CollabAgentStatusInterrupted CollabAgentStatus = "interrupted"
	CollabAgentStatusCompleted   CollabAgentStatus = "completed"
	CollabAgentStatusErrored     CollabAgentStatus = "errored"
	CollabAgentStatusShutdown    CollabAgentStatus = "shutdown"
	CollabAgentStatusNotFound    CollabAgentStatus = "notFound"
)

type CollabAgentTool string

const (
	CollabAgentToolSpawnAgent  CollabAgentTool = "spawnAgent"
	CollabAgentToolSendInput   CollabAgentTool = "sendInput"
	CollabAgentToolResumeAgent CollabAgentTool = "resumeAgent"
	CollabAgentToolWait        CollabAgentTool = "wait"
	CollabAgentToolCloseAgent  CollabAgentTool = "closeAgent"
)

type CollabAgentToolCallStatus string

const (
	CollabAgentToolCallStatusInProgress CollabAgentToolCallStatus = "inProgress"
	CollabAgentToolCallStatusCompleted  CollabAgentToolCallStatus = "completed"
	CollabAgentToolCallStatusFailed     CollabAgentToolCallStatus = "failed"
)

type CollaborationMode struct {
	Mode     ModeKind `json:"mode"`
	Settings Settings `json:"settings"`
}

type ModeKind string

const (
	ModeKindPlan    ModeKind = "plan"
	ModeKindDefault ModeKind = "default"
)

type Settings struct {
	DeveloperInstructions *string          `json:"developer_instructions,omitempty"`
	Model                 string           `json:"model"`
	ReasoningEffort       *ReasoningEffort `json:"reasoning_effort,omitempty"`
}

type CollaborationModeMask struct {
	Mode            *ModeKind        `json:"mode,omitempty"`
	Model           *string          `json:"model,omitempty"`
	Name            string           `json:"name"`
	ReasoningEffort *ReasoningEffort `json:"reasoning_effort,omitempty"`
}

type CommandAction struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *CommandAction) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "name", "path", "type")) {
					break
				}
				var v ReadCommandAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "listFiles":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "type")) {
					break
				}
				var v ListFilesCommandAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "search":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "type")) {
					break
				}
				var v SearchCommandAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "unknown":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "type")) {
					break
				}
				var v UnknownCommandAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "name", "path", "type") {
		var v ReadCommandAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "type") {
		var v ListFilesCommandAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "type") {
		var v SearchCommandAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "type") {
		var v UnknownCommandAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u CommandAction) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ReadCommandAction struct {
	Command string              `json:"command"`
	Name    string              `json:"name"`
	Path    LegacyAppPathString `json:"path"`
	Type    string              `json:"type"`
}

type ListFilesCommandAction struct {
	Command string  `json:"command"`
	Path    *string `json:"path,omitempty"`
	Type    string  `json:"type"`
}

type SearchCommandAction struct {
	Command string  `json:"command"`
	Path    *string `json:"path,omitempty"`
	Query   *string `json:"query,omitempty"`
	Type    string  `json:"type"`
}

type UnknownCommandAction struct {
	Command string `json:"command"`
	Type    string `json:"type"`
}

type CommandExecOutputDeltaNotification struct {
	CapReached  bool                    `json:"capReached"`
	DeltaBase64 string                  `json:"deltaBase64"`
	ProcessID   string                  `json:"processId"`
	Stream      CommandExecOutputStream `json:"stream"`
}

type CommandExecOutputStream struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *CommandExecOutputStream) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "stdout") {
		var v CommandExecOutputStreamVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "stderr") {
		var v CommandExecOutputStreamVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u CommandExecOutputStream) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type CommandExecOutputStreamVariant1 string

const (
	CommandExecOutputStreamVariant1Stdout CommandExecOutputStreamVariant1 = "stdout"
)

type CommandExecOutputStreamVariant2 string

const (
	CommandExecOutputStreamVariant2Stderr CommandExecOutputStreamVariant2 = "stderr"
)

type CommandExecResizeResponse struct {
}

type CommandExecResponse struct {
	ExitCode int64  `json:"exitCode"`
	Stderr   string `json:"stderr"`
	Stdout   string `json:"stdout"`
}

type CommandExecTerminateResponse struct {
}

type CommandExecWriteResponse struct {
}

type CommandExecutionApprovalDecision struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *CommandExecutionApprovalDecision) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "accept") {
		var v CommandExecutionApprovalDecisionVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "acceptForSession") {
		var v CommandExecutionApprovalDecisionVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "acceptWithExecpolicyAmendment") {
		var v AcceptWithExecpolicyAmendmentCommandExecutionApprovalDecision
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "applyNetworkPolicyAmendment") {
		var v ApplyNetworkPolicyAmendmentCommandExecutionApprovalDecision
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "decline") {
		var v CommandExecutionApprovalDecisionVariant5
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "cancel") {
		var v CommandExecutionApprovalDecisionVariant6
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u CommandExecutionApprovalDecision) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type CommandExecutionApprovalDecisionVariant1 string

const (
	CommandExecutionApprovalDecisionVariant1Accept CommandExecutionApprovalDecisionVariant1 = "accept"
)

type CommandExecutionApprovalDecisionVariant2 string

const (
	CommandExecutionApprovalDecisionVariant2AcceptForSession CommandExecutionApprovalDecisionVariant2 = "acceptForSession"
)

type AcceptWithExecpolicyAmendmentCommandExecutionApprovalDecision struct {
	AcceptWithExecpolicyAmendment map[string]any `json:"acceptWithExecpolicyAmendment"`
}

type ApplyNetworkPolicyAmendmentCommandExecutionApprovalDecision struct {
	ApplyNetworkPolicyAmendment map[string]any `json:"applyNetworkPolicyAmendment"`
}

type CommandExecutionApprovalDecisionVariant5 string

const (
	CommandExecutionApprovalDecisionVariant5Decline CommandExecutionApprovalDecisionVariant5 = "decline"
)

type CommandExecutionApprovalDecisionVariant6 string

const (
	CommandExecutionApprovalDecisionVariant6Cancel CommandExecutionApprovalDecisionVariant6 = "cancel"
)

type CommandExecutionOutputDeltaNotification struct {
	Delta    string `json:"delta"`
	ItemID   string `json:"itemId"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type CommandExecutionRequestApprovalParams struct {
	ApprovalID                      *string                  `json:"approvalId,omitempty"`
	Command                         *string                  `json:"command,omitempty"`
	CommandActions                  []CommandAction          `json:"commandActions,omitempty"`
	Cwd                             *LegacyAppPathString     `json:"cwd,omitempty"`
	EnvironmentID                   *string                  `json:"environmentId,omitempty"`
	ItemID                          string                   `json:"itemId"`
	NetworkApprovalContext          *NetworkApprovalContext  `json:"networkApprovalContext,omitempty"`
	ProposedExecpolicyAmendment     []string                 `json:"proposedExecpolicyAmendment,omitempty"`
	ProposedNetworkPolicyAmendments []NetworkPolicyAmendment `json:"proposedNetworkPolicyAmendments,omitempty"`
	Reason                          *string                  `json:"reason,omitempty"`
	StartedAtMs                     int64                    `json:"startedAtMs"`
	ThreadID                        string                   `json:"threadId"`
	TurnID                          string                   `json:"turnId"`
}

type NetworkApprovalContext struct {
	Host     string                  `json:"host"`
	Protocol NetworkApprovalProtocol `json:"protocol"`
}

type NetworkApprovalProtocol string

const (
	NetworkApprovalProtocolHttp      NetworkApprovalProtocol = "http"
	NetworkApprovalProtocolHttps     NetworkApprovalProtocol = "https"
	NetworkApprovalProtocolSocks5Tcp NetworkApprovalProtocol = "socks5Tcp"
	NetworkApprovalProtocolSocks5Udp NetworkApprovalProtocol = "socks5Udp"
)

type NetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

type NetworkPolicyRuleAction string

const (
	NetworkPolicyRuleActionAllow NetworkPolicyRuleAction = "allow"
	NetworkPolicyRuleActionDeny  NetworkPolicyRuleAction = "deny"
)

type CommandExecutionRequestApprovalResponse struct {
	Decision CommandExecutionApprovalDecision `json:"decision"`
}

type CommandExecutionSource string

const (
	CommandExecutionSourceAgent                  CommandExecutionSource = "agent"
	CommandExecutionSourceUserShell              CommandExecutionSource = "userShell"
	CommandExecutionSourceUnifiedExecStartup     CommandExecutionSource = "unifiedExecStartup"
	CommandExecutionSourceUnifiedExecInteraction CommandExecutionSource = "unifiedExecInteraction"
)

type CommandExecutionStatus string

const (
	CommandExecutionStatusInProgress CommandExecutionStatus = "inProgress"
	CommandExecutionStatusCompleted  CommandExecutionStatus = "completed"
	CommandExecutionStatusFailed     CommandExecutionStatus = "failed"
	CommandExecutionStatusDeclined   CommandExecutionStatus = "declined"
)

type ComputerUseRequirements struct {
	AllowLockedComputerUse *bool `json:"allowLockedComputerUse,omitempty"`
}

type Config struct {
	Analytics                       *AnalyticsConfig            `json:"analytics,omitempty"`
	ApprovalPolicy                  *AskForApproval             `json:"approval_policy,omitempty"`
	ApprovalsReviewer               *ApprovalsReviewer          `json:"approvals_reviewer,omitempty"`
	CompactPrompt                   *string                     `json:"compact_prompt,omitempty"`
	Desktop                         map[string]any              `json:"desktop,omitempty"`
	DeveloperInstructions           *string                     `json:"developer_instructions,omitempty"`
	ForcedChatgptWorkspaceID        *ForcedChatgptWorkspaceIds  `json:"forced_chatgpt_workspace_id,omitempty"`
	ForcedLoginMethod               *ForcedLoginMethod          `json:"forced_login_method,omitempty"`
	Instructions                    *string                     `json:"instructions,omitempty"`
	Model                           *string                     `json:"model,omitempty"`
	ModelAutoCompactTokenLimit      *int64                      `json:"model_auto_compact_token_limit,omitempty"`
	ModelAutoCompactTokenLimitScope *AutoCompactTokenLimitScope `json:"model_auto_compact_token_limit_scope,omitempty"`
	ModelContextWindow              *int64                      `json:"model_context_window,omitempty"`
	ModelProvider                   *string                     `json:"model_provider,omitempty"`
	ModelReasoningEffort            *ReasoningEffort            `json:"model_reasoning_effort,omitempty"`
	ModelReasoningSummary           *ReasoningSummary           `json:"model_reasoning_summary,omitempty"`
	ModelVerbosity                  *Verbosity                  `json:"model_verbosity,omitempty"`
	ReviewModel                     *string                     `json:"review_model,omitempty"`
	SandboxMode                     *SandboxMode                `json:"sandbox_mode,omitempty"`
	SandboxWorkspaceWrite           *SandboxWorkspaceWrite      `json:"sandbox_workspace_write,omitempty"`
	ServiceTier                     *string                     `json:"service_tier,omitempty"`
	Tools                           *ToolsV2                    `json:"tools,omitempty"`
	WebSearch                       *WebSearchMode              `json:"web_search,omitempty"`
	Extras                          map[string]json.RawMessage  `json:"-"`
}

func (v *Config) UnmarshalJSON(data []byte) error {
	type ConfigAlias struct {
		Analytics                       *AnalyticsConfig            `json:"analytics,omitempty"`
		ApprovalPolicy                  *AskForApproval             `json:"approval_policy,omitempty"`
		ApprovalsReviewer               *ApprovalsReviewer          `json:"approvals_reviewer,omitempty"`
		CompactPrompt                   *string                     `json:"compact_prompt,omitempty"`
		Desktop                         map[string]any              `json:"desktop,omitempty"`
		DeveloperInstructions           *string                     `json:"developer_instructions,omitempty"`
		ForcedChatgptWorkspaceID        *ForcedChatgptWorkspaceIds  `json:"forced_chatgpt_workspace_id,omitempty"`
		ForcedLoginMethod               *ForcedLoginMethod          `json:"forced_login_method,omitempty"`
		Instructions                    *string                     `json:"instructions,omitempty"`
		Model                           *string                     `json:"model,omitempty"`
		ModelAutoCompactTokenLimit      *int64                      `json:"model_auto_compact_token_limit,omitempty"`
		ModelAutoCompactTokenLimitScope *AutoCompactTokenLimitScope `json:"model_auto_compact_token_limit_scope,omitempty"`
		ModelContextWindow              *int64                      `json:"model_context_window,omitempty"`
		ModelProvider                   *string                     `json:"model_provider,omitempty"`
		ModelReasoningEffort            *ReasoningEffort            `json:"model_reasoning_effort,omitempty"`
		ModelReasoningSummary           *ReasoningSummary           `json:"model_reasoning_summary,omitempty"`
		ModelVerbosity                  *Verbosity                  `json:"model_verbosity,omitempty"`
		ReviewModel                     *string                     `json:"review_model,omitempty"`
		SandboxMode                     *SandboxMode                `json:"sandbox_mode,omitempty"`
		SandboxWorkspaceWrite           *SandboxWorkspaceWrite      `json:"sandbox_workspace_write,omitempty"`
		ServiceTier                     *string                     `json:"service_tier,omitempty"`
		Tools                           *ToolsV2                    `json:"tools,omitempty"`
		WebSearch                       *WebSearchMode              `json:"web_search,omitempty"`
	}
	var aux ConfigAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	v.Analytics = aux.Analytics
	v.ApprovalPolicy = aux.ApprovalPolicy
	v.ApprovalsReviewer = aux.ApprovalsReviewer
	v.CompactPrompt = aux.CompactPrompt
	v.Desktop = aux.Desktop
	v.DeveloperInstructions = aux.DeveloperInstructions
	v.ForcedChatgptWorkspaceID = aux.ForcedChatgptWorkspaceID
	v.ForcedLoginMethod = aux.ForcedLoginMethod
	v.Instructions = aux.Instructions
	v.Model = aux.Model
	v.ModelAutoCompactTokenLimit = aux.ModelAutoCompactTokenLimit
	v.ModelAutoCompactTokenLimitScope = aux.ModelAutoCompactTokenLimitScope
	v.ModelContextWindow = aux.ModelContextWindow
	v.ModelProvider = aux.ModelProvider
	v.ModelReasoningEffort = aux.ModelReasoningEffort
	v.ModelReasoningSummary = aux.ModelReasoningSummary
	v.ModelVerbosity = aux.ModelVerbosity
	v.ReviewModel = aux.ReviewModel
	v.SandboxMode = aux.SandboxMode
	v.SandboxWorkspaceWrite = aux.SandboxWorkspaceWrite
	v.ServiceTier = aux.ServiceTier
	v.Tools = aux.Tools
	v.WebSearch = aux.WebSearch
	var extras map[string]json.RawMessage
	if err := json.Unmarshal(data, &extras); err != nil {
		return err
	}
	delete(extras, "analytics")
	delete(extras, "approval_policy")
	delete(extras, "approvals_reviewer")
	delete(extras, "compact_prompt")
	delete(extras, "desktop")
	delete(extras, "developer_instructions")
	delete(extras, "forced_chatgpt_workspace_id")
	delete(extras, "forced_login_method")
	delete(extras, "instructions")
	delete(extras, "model")
	delete(extras, "model_auto_compact_token_limit")
	delete(extras, "model_auto_compact_token_limit_scope")
	delete(extras, "model_context_window")
	delete(extras, "model_provider")
	delete(extras, "model_reasoning_effort")
	delete(extras, "model_reasoning_summary")
	delete(extras, "model_verbosity")
	delete(extras, "review_model")
	delete(extras, "sandbox_mode")
	delete(extras, "sandbox_workspace_write")
	delete(extras, "service_tier")
	delete(extras, "tools")
	delete(extras, "web_search")
	if len(extras) == 0 {
		v.Extras = nil
	} else {
		v.Extras = extras
	}
	return nil
}

func (v Config) MarshalJSON() ([]byte, error) {
	type ConfigAlias struct {
		Analytics                       *AnalyticsConfig            `json:"analytics,omitempty"`
		ApprovalPolicy                  *AskForApproval             `json:"approval_policy,omitempty"`
		ApprovalsReviewer               *ApprovalsReviewer          `json:"approvals_reviewer,omitempty"`
		CompactPrompt                   *string                     `json:"compact_prompt,omitempty"`
		Desktop                         map[string]any              `json:"desktop,omitempty"`
		DeveloperInstructions           *string                     `json:"developer_instructions,omitempty"`
		ForcedChatgptWorkspaceID        *ForcedChatgptWorkspaceIds  `json:"forced_chatgpt_workspace_id,omitempty"`
		ForcedLoginMethod               *ForcedLoginMethod          `json:"forced_login_method,omitempty"`
		Instructions                    *string                     `json:"instructions,omitempty"`
		Model                           *string                     `json:"model,omitempty"`
		ModelAutoCompactTokenLimit      *int64                      `json:"model_auto_compact_token_limit,omitempty"`
		ModelAutoCompactTokenLimitScope *AutoCompactTokenLimitScope `json:"model_auto_compact_token_limit_scope,omitempty"`
		ModelContextWindow              *int64                      `json:"model_context_window,omitempty"`
		ModelProvider                   *string                     `json:"model_provider,omitempty"`
		ModelReasoningEffort            *ReasoningEffort            `json:"model_reasoning_effort,omitempty"`
		ModelReasoningSummary           *ReasoningSummary           `json:"model_reasoning_summary,omitempty"`
		ModelVerbosity                  *Verbosity                  `json:"model_verbosity,omitempty"`
		ReviewModel                     *string                     `json:"review_model,omitempty"`
		SandboxMode                     *SandboxMode                `json:"sandbox_mode,omitempty"`
		SandboxWorkspaceWrite           *SandboxWorkspaceWrite      `json:"sandbox_workspace_write,omitempty"`
		ServiceTier                     *string                     `json:"service_tier,omitempty"`
		Tools                           *ToolsV2                    `json:"tools,omitempty"`
		WebSearch                       *WebSearchMode              `json:"web_search,omitempty"`
	}
	aux := ConfigAlias{
		Analytics:                       v.Analytics,
		ApprovalPolicy:                  v.ApprovalPolicy,
		ApprovalsReviewer:               v.ApprovalsReviewer,
		CompactPrompt:                   v.CompactPrompt,
		Desktop:                         v.Desktop,
		DeveloperInstructions:           v.DeveloperInstructions,
		ForcedChatgptWorkspaceID:        v.ForcedChatgptWorkspaceID,
		ForcedLoginMethod:               v.ForcedLoginMethod,
		Instructions:                    v.Instructions,
		Model:                           v.Model,
		ModelAutoCompactTokenLimit:      v.ModelAutoCompactTokenLimit,
		ModelAutoCompactTokenLimitScope: v.ModelAutoCompactTokenLimitScope,
		ModelContextWindow:              v.ModelContextWindow,
		ModelProvider:                   v.ModelProvider,
		ModelReasoningEffort:            v.ModelReasoningEffort,
		ModelReasoningSummary:           v.ModelReasoningSummary,
		ModelVerbosity:                  v.ModelVerbosity,
		ReviewModel:                     v.ReviewModel,
		SandboxMode:                     v.SandboxMode,
		SandboxWorkspaceWrite:           v.SandboxWorkspaceWrite,
		ServiceTier:                     v.ServiceTier,
		Tools:                           v.Tools,
		WebSearch:                       v.WebSearch,
	}
	encoded, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, err
	}
	for key, value := range v.Extras {
		if _, exists := payload[key]; exists {
			continue
		}
		payload[key] = append(json.RawMessage(nil), value...)
	}
	return json.Marshal(payload)
}

type ForcedChatgptWorkspaceIds struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ForcedChatgptWorkspaceIds) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksString(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksArray(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ForcedChatgptWorkspaceIds) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ForcedLoginMethod string

const (
	ForcedLoginMethodChatgpt ForcedLoginMethod = "chatgpt"
	ForcedLoginMethodApi     ForcedLoginMethod = "api"
)

type Verbosity string

const (
	VerbosityLow    Verbosity = "low"
	VerbosityMedium Verbosity = "medium"
	VerbosityHigh   Verbosity = "high"
)

type SandboxWorkspaceWrite struct {
	ExcludeSlashTmp     *bool    `json:"exclude_slash_tmp,omitempty"`
	ExcludeTmpdirEnvVar *bool    `json:"exclude_tmpdir_env_var,omitempty"`
	NetworkAccess       *bool    `json:"network_access,omitempty"`
	WritableRoots       []string `json:"writable_roots,omitempty"`
}

type ToolsV2 struct {
	WebSearch *WebSearchToolConfig `json:"web_search,omitempty"`
}

type WebSearchToolConfig struct {
	AllowedDomains []string              `json:"allowed_domains,omitempty"`
	ContextSize    *WebSearchContextSize `json:"context_size,omitempty"`
	Location       *WebSearchLocation    `json:"location,omitempty"`
}

type WebSearchContextSize string

const (
	WebSearchContextSizeLow    WebSearchContextSize = "low"
	WebSearchContextSizeMedium WebSearchContextSize = "medium"
	WebSearchContextSizeHigh   WebSearchContextSize = "high"
)

type WebSearchLocation struct {
	City     *string `json:"city,omitempty"`
	Country  *string `json:"country,omitempty"`
	Region   *string `json:"region,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

type WebSearchMode string

const (
	WebSearchModeDisabled WebSearchMode = "disabled"
	WebSearchModeCached   WebSearchMode = "cached"
	WebSearchModeIndexed  WebSearchMode = "indexed"
	WebSearchModeLive     WebSearchMode = "live"
)

type ConfigLayer struct {
	Config         any               `json:"config"`
	DisabledReason *string           `json:"disabledReason,omitempty"`
	Name           ConfigLayerSource `json:"name"`
	Version        string            `json:"version"`
}

type ConfigLayerSource struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ConfigLayerSource) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "mdm":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "domain", "key", "type")) {
					break
				}
				var v MdmConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "system":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "file", "type")) {
					break
				}
				var v SystemConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "enterpriseManaged":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "name", "type")) {
					break
				}
				var v EnterpriseManagedConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "user":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "file", "type")) {
					break
				}
				var v UserConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "project":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "dotCodexFolder", "type")) {
					break
				}
				var v ProjectConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "sessionFlags":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v SessionFlagsConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "legacyManagedConfigTomlFromFile":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "file", "type")) {
					break
				}
				var v LegacyManagedConfigTomlFromFileConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "legacyManagedConfigTomlFromMdm":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v LegacyManagedConfigTomlFromMdmConfigLayerSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "domain", "key", "type") {
		var v MdmConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "file", "type") {
		var v SystemConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "name", "type") {
		var v EnterpriseManagedConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "file", "type") {
		var v UserConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "dotCodexFolder", "type") {
		var v ProjectConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v SessionFlagsConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "file", "type") {
		var v LegacyManagedConfigTomlFromFileConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v LegacyManagedConfigTomlFromMdmConfigLayerSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ConfigLayerSource) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type MdmConfigLayerSource struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
	Type   string `json:"type"`
}

type SystemConfigLayerSource struct {
	File AbsolutePathBuf `json:"file"`
	Type string          `json:"type"`
}

type EnterpriseManagedConfigLayerSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type UserConfigLayerSource struct {
	File    AbsolutePathBuf `json:"file"`
	Profile *string         `json:"profile,omitempty"`
	Type    string          `json:"type"`
}

type ProjectConfigLayerSource struct {
	DotCodexFolder AbsolutePathBuf `json:"dotCodexFolder"`
	Type           string          `json:"type"`
}

type SessionFlagsConfigLayerSource struct {
	Type string `json:"type"`
}

type LegacyManagedConfigTomlFromFileConfigLayerSource struct {
	File AbsolutePathBuf `json:"file"`
	Type string          `json:"type"`
}

type LegacyManagedConfigTomlFromMdmConfigLayerSource struct {
	Type string `json:"type"`
}

type ConfigLayerMetadata struct {
	Name    ConfigLayerSource `json:"name"`
	Version string            `json:"version"`
}

type ConfigReadResponse struct {
	Config  Config                         `json:"config"`
	Layers  []ConfigLayer                  `json:"layers,omitempty"`
	Origins map[string]ConfigLayerMetadata `json:"origins"`
}

type ConfigRequirements struct {
	AllowAppshots                        *bool                     `json:"allowAppshots,omitempty"`
	AllowLoginShell                      *bool                     `json:"allowLoginShell,omitempty"`
	AllowManagedHooksOnly                *bool                     `json:"allowManagedHooksOnly,omitempty"`
	AllowRemoteControl                   *bool                     `json:"allowRemoteControl,omitempty"`
	AllowedApprovalPolicies              []AskForApproval          `json:"allowedApprovalPolicies,omitempty"`
	AllowedPermissionProfiles            map[string]bool           `json:"allowedPermissionProfiles,omitempty"`
	AllowedSandboxModes                  []SandboxMode             `json:"allowedSandboxModes,omitempty"`
	AllowedWebSearchModes                []WebSearchMode           `json:"allowedWebSearchModes,omitempty"`
	AllowedWindowsSandboxImplementations []WindowsSandboxSetupMode `json:"allowedWindowsSandboxImplementations,omitempty"`
	BrowserUse                           *BrowserUseRequirements   `json:"browserUse,omitempty"`
	CheckForUpdateOnStartup              *bool                     `json:"checkForUpdateOnStartup,omitempty"`
	ComputerUse                          *ComputerUseRequirements  `json:"computerUse,omitempty"`
	DefaultPermissions                   *string                   `json:"defaultPermissions,omitempty"`
	EnforceResidency                     *ResidencyRequirement     `json:"enforceResidency,omitempty"`
	FeatureRequirements                  map[string]bool           `json:"featureRequirements,omitempty"`
	Feedback                             *FeedbackRequirements     `json:"feedback,omitempty"`
	LogDir                               *string                   `json:"logDir,omitempty"`
	ModelCatalogJson                     *string                   `json:"modelCatalogJson,omitempty"`
	Models                               *ModelsRequirements       `json:"models,omitempty"`
	SqliteHome                           *string                   `json:"sqliteHome,omitempty"`
	WindowsSandboxPrivateDesktop         *bool                     `json:"windowsSandboxPrivateDesktop,omitempty"`
}

type ResidencyRequirement string

const (
	ResidencyRequirementUs ResidencyRequirement = "us"
)

type FeedbackRequirements struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type ModelsRequirements struct {
	NewThread *NewThreadModelDefaults `json:"newThread,omitempty"`
}

type NewThreadModelDefaults struct {
	Model                *string          `json:"model,omitempty"`
	ModelReasoningEffort *ReasoningEffort `json:"modelReasoningEffort,omitempty"`
	ServiceTier          *string          `json:"serviceTier,omitempty"`
}

type ConfigRequirementsReadResponse struct {
	Requirements *ConfigRequirements `json:"requirements,omitempty"`
}

type ConfigWarningNotification struct {
	Details *string    `json:"details,omitempty"`
	Path    *string    `json:"path,omitempty"`
	Range   *TextRange `json:"range,omitempty"`
	Summary string     `json:"summary"`
}

type TextRange struct {
	End   TextPosition `json:"end"`
	Start TextPosition `json:"start"`
}

type TextPosition struct {
	Column int64 `json:"column"`
	Line   int64 `json:"line"`
}

type ConfigWriteResponse struct {
	FilePath           AbsolutePathBuf     `json:"filePath"`
	OverriddenMetadata *OverriddenMetadata `json:"overriddenMetadata,omitempty"`
	Status             WriteStatus         `json:"status"`
	Version            string              `json:"version"`
}

type OverriddenMetadata struct {
	EffectiveValue  any                 `json:"effectiveValue"`
	Message         string              `json:"message"`
	OverridingLayer ConfigLayerMetadata `json:"overridingLayer"`
}

type WriteStatus string

const (
	WriteStatusOk           WriteStatus = "ok"
	WriteStatusOkOverridden WriteStatus = "okOverridden"
)

type ConfiguredHookHandler struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ConfiguredHookHandler) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "command":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "async", "command", "type")) {
					break
				}
				var v CommandConfiguredHookHandler
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "prompt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v PromptConfiguredHookHandler
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "agent":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v AgentConfiguredHookHandler
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "async", "command", "type") {
		var v CommandConfiguredHookHandler
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v PromptConfiguredHookHandler
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v AgentConfiguredHookHandler
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ConfiguredHookHandler) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type CommandConfiguredHookHandler struct {
	AdditionalContextLimit *int64  `json:"additionalContextLimit,omitempty"`
	Async                  bool    `json:"async"`
	Command                string  `json:"command"`
	CommandWindows         *string `json:"commandWindows,omitempty"`
	StatusMessage          *string `json:"statusMessage,omitempty"`
	TimeoutSec             *int64  `json:"timeoutSec,omitempty"`
	Type                   string  `json:"type"`
}

type PromptConfiguredHookHandler struct {
	Type string `json:"type"`
}

type AgentConfiguredHookHandler struct {
	Type string `json:"type"`
}

type ConfiguredHookMatcherGroup struct {
	Hooks   []ConfiguredHookHandler `json:"hooks"`
	Matcher *string                 `json:"matcher,omitempty"`
}

type ConsumeAccountRateLimitResetCreditOutcome struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ConsumeAccountRateLimitResetCreditOutcome) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "reset") {
		var v ConsumeAccountRateLimitResetCreditOutcomeVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "nothingToReset") {
		var v ConsumeAccountRateLimitResetCreditOutcomeVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "noCredit") {
		var v ConsumeAccountRateLimitResetCreditOutcomeVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "alreadyRedeemed") {
		var v ConsumeAccountRateLimitResetCreditOutcomeVariant4
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ConsumeAccountRateLimitResetCreditOutcome) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ConsumeAccountRateLimitResetCreditOutcomeVariant1 string

const (
	ConsumeAccountRateLimitResetCreditOutcomeVariant1Reset ConsumeAccountRateLimitResetCreditOutcomeVariant1 = "reset"
)

type ConsumeAccountRateLimitResetCreditOutcomeVariant2 string

const (
	ConsumeAccountRateLimitResetCreditOutcomeVariant2NothingToReset ConsumeAccountRateLimitResetCreditOutcomeVariant2 = "nothingToReset"
)

type ConsumeAccountRateLimitResetCreditOutcomeVariant3 string

const (
	ConsumeAccountRateLimitResetCreditOutcomeVariant3NoCredit ConsumeAccountRateLimitResetCreditOutcomeVariant3 = "noCredit"
)

type ConsumeAccountRateLimitResetCreditOutcomeVariant4 string

const (
	ConsumeAccountRateLimitResetCreditOutcomeVariant4AlreadyRedeemed ConsumeAccountRateLimitResetCreditOutcomeVariant4 = "alreadyRedeemed"
)

type ConsumeAccountRateLimitResetCreditResponse struct {
	Outcome ConsumeAccountRateLimitResetCreditOutcome `json:"outcome"`
}

type ContentItem struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ContentItem) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "input_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v InputTextContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "input_image":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "image_url", "type")) {
					break
				}
				var v InputImageContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "input_audio":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "audio_url", "type")) {
					break
				}
				var v InputAudioContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "output_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v OutputTextContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v InputTextContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "image_url", "type") {
		var v InputImageContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "audio_url", "type") {
		var v InputAudioContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v OutputTextContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ContentItem) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InputTextContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type InputImageContentItem struct {
	Detail   *ImageDetail `json:"detail,omitempty"`
	ImageUrl string       `json:"image_url"`
	Type     string       `json:"type"`
}

type InputAudioContentItem struct {
	AudioUrl string `json:"audio_url"`
	Type     string `json:"type"`
}

type OutputTextContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type ContextCompactedNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type ConversationTextRole string

const (
	ConversationTextRoleUser      ConversationTextRole = "user"
	ConversationTextRoleDeveloper ConversationTextRole = "developer"
	ConversationTextRoleAssistant ConversationTextRole = "assistant"
)

type DeprecationNoticeNotification struct {
	Details *string `json:"details,omitempty"`
	Summary string  `json:"summary"`
}

type DynamicToolCallOutputContentItem struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *DynamicToolCallOutputContentItem) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "inputText":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v InputTextDynamicToolCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "inputImage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "imageUrl", "type")) {
					break
				}
				var v InputImageDynamicToolCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "inputAudio":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "audioUrl", "type")) {
					break
				}
				var v InputAudioDynamicToolCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v InputTextDynamicToolCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "imageUrl", "type") {
		var v InputImageDynamicToolCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "audioUrl", "type") {
		var v InputAudioDynamicToolCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u DynamicToolCallOutputContentItem) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InputTextDynamicToolCallOutputContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type InputImageDynamicToolCallOutputContentItem struct {
	ImageUrl string `json:"imageUrl"`
	Type     string `json:"type"`
}

type InputAudioDynamicToolCallOutputContentItem struct {
	AudioUrl string `json:"audioUrl"`
	Type     string `json:"type"`
}

type DynamicToolCallParams struct {
	Arguments any     `json:"arguments"`
	CallID    string  `json:"callId"`
	Namespace *string `json:"namespace,omitempty"`
	ThreadID  string  `json:"threadId"`
	Tool      string  `json:"tool"`
	TurnID    string  `json:"turnId"`
}

type DynamicToolCallResponse struct {
	ContentItems []DynamicToolCallOutputContentItem `json:"contentItems"`
	Success      bool                               `json:"success"`
}

type DynamicToolCallStatus string

const (
	DynamicToolCallStatusInProgress DynamicToolCallStatus = "inProgress"
	DynamicToolCallStatusCompleted  DynamicToolCallStatus = "completed"
	DynamicToolCallStatusFailed     DynamicToolCallStatus = "failed"
)

type DynamicToolNamespaceTool struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *DynamicToolNamespaceTool) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "function":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "description", "inputSchema", "name", "type")) {
					break
				}
				var v FunctionDynamicToolNamespaceTool
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "description", "inputSchema", "name", "type") {
		var v FunctionDynamicToolNamespaceTool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u DynamicToolNamespaceTool) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type FunctionDynamicToolNamespaceTool struct {
	DeferLoading *bool  `json:"deferLoading,omitempty"`
	Description  string `json:"description"`
	InputSchema  any    `json:"inputSchema"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type DynamicToolSpec struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *DynamicToolSpec) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "function":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "description", "inputSchema", "name", "type")) {
					break
				}
				var v FunctionDynamicToolSpec
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "namespace":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "description", "name", "tools", "type")) {
					break
				}
				var v NamespaceDynamicToolSpec
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "description", "inputSchema", "name", "type") {
		var v FunctionDynamicToolSpec
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "description", "name", "tools", "type") {
		var v NamespaceDynamicToolSpec
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u DynamicToolSpec) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type FunctionDynamicToolSpec struct {
	DeferLoading *bool  `json:"deferLoading,omitempty"`
	Description  string `json:"description"`
	InputSchema  any    `json:"inputSchema"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type NamespaceDynamicToolSpec struct {
	Description string                     `json:"description"`
	Name        string                     `json:"name"`
	Tools       []DynamicToolNamespaceTool `json:"tools"`
	Type        string                     `json:"type"`
}

type EnvironmentConnectionNotification struct {
	EnvironmentID string `json:"environmentId"`
	ThreadID      string `json:"threadId"`
}

type ErrorNotification struct {
	Error     TurnError `json:"error"`
	ThreadID  string    `json:"threadId"`
	TurnID    string    `json:"turnId"`
	WillRetry bool      `json:"willRetry"`
}

type TurnError struct {
	AdditionalDetails *string         `json:"additionalDetails,omitempty"`
	CodexErrorInfo    *CodexErrorInfo `json:"codexErrorInfo,omitempty"`
	Message           string          `json:"message"`
}

type ExecCommandApprovalParams struct {
	ApprovalID     *string         `json:"approvalId,omitempty"`
	CallID         string          `json:"callId"`
	Command        []string        `json:"command"`
	ConversationID ThreadId        `json:"conversationId"`
	Cwd            string          `json:"cwd"`
	ParsedCmd      []ParsedCommand `json:"parsedCmd"`
	Reason         *string         `json:"reason,omitempty"`
}

type ParsedCommand struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ParsedCommand) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "read":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "cmd", "name", "path", "type")) {
					break
				}
				var v ReadParsedCommand
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "list_files":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type")) {
					break
				}
				var v ListFilesParsedCommand
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "search":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type")) {
					break
				}
				var v SearchParsedCommand
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "unknown":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type")) {
					break
				}
				var v UnknownParsedCommand
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "cmd", "name", "path", "type") {
		var v ReadParsedCommand
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type") {
		var v ListFilesParsedCommand
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type") {
		var v SearchParsedCommand
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "cmd", "type") {
		var v UnknownParsedCommand
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ParsedCommand) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ReadParsedCommand struct {
	Cmd  string `json:"cmd"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type ListFilesParsedCommand struct {
	Cmd  string  `json:"cmd"`
	Path *string `json:"path,omitempty"`
	Type string  `json:"type"`
}

type SearchParsedCommand struct {
	Cmd   string  `json:"cmd"`
	Path  *string `json:"path,omitempty"`
	Query *string `json:"query,omitempty"`
	Type  string  `json:"type"`
}

type UnknownParsedCommand struct {
	Cmd  string `json:"cmd"`
	Type string `json:"type"`
}

type ExecCommandApprovalResponse struct {
	Decision ReviewDecision `json:"decision"`
}

type ExperimentalFeature struct {
	Announcement   *string                  `json:"announcement,omitempty"`
	DefaultEnabled bool                     `json:"defaultEnabled"`
	Description    *string                  `json:"description,omitempty"`
	DisplayName    *string                  `json:"displayName,omitempty"`
	Enabled        bool                     `json:"enabled"`
	Name           string                   `json:"name"`
	Stage          ExperimentalFeatureStage `json:"stage"`
}

type ExperimentalFeatureStage struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ExperimentalFeatureStage) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "beta") {
		var v ExperimentalFeatureStageVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "underDevelopment") {
		var v ExperimentalFeatureStageVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "stable") {
		var v ExperimentalFeatureStageVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "deprecated") {
		var v ExperimentalFeatureStageVariant4
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "removed") {
		var v ExperimentalFeatureStageVariant5
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ExperimentalFeatureStage) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ExperimentalFeatureStageVariant1 string

const (
	ExperimentalFeatureStageVariant1Beta ExperimentalFeatureStageVariant1 = "beta"
)

type ExperimentalFeatureStageVariant2 string

const (
	ExperimentalFeatureStageVariant2UnderDevelopment ExperimentalFeatureStageVariant2 = "underDevelopment"
)

type ExperimentalFeatureStageVariant3 string

const (
	ExperimentalFeatureStageVariant3Stable ExperimentalFeatureStageVariant3 = "stable"
)

type ExperimentalFeatureStageVariant4 string

const (
	ExperimentalFeatureStageVariant4Deprecated ExperimentalFeatureStageVariant4 = "deprecated"
)

type ExperimentalFeatureStageVariant5 string

const (
	ExperimentalFeatureStageVariant5Removed ExperimentalFeatureStageVariant5 = "removed"
)

type ExperimentalFeatureEnablementSetResponse struct {
	Enablement map[string]bool `json:"enablement"`
}

type ExperimentalFeatureListResponse struct {
	Data       []ExperimentalFeature `json:"data"`
	NextCursor *string               `json:"nextCursor,omitempty"`
}

type ExternalAgentConfigDetectResponse struct {
	Connectors []ExternalAgentDetectedConnectorCandidate `json:"connectors,omitempty"`
	Items      []ExternalAgentConfigMigrationItem        `json:"items"`
}

type ExternalAgentDetectedConnectorCandidate struct {
	Name         string                               `json:"name"`
	SessionCount int64                                `json:"sessionCount"`
	Source       ExternalAgentDetectedConnectorSource `json:"source"`
}

type ExternalAgentDetectedConnectorSource string

const (
	ExternalAgentDetectedConnectorSourceRemoteMcpServersConfig ExternalAgentDetectedConnectorSource = "remoteMcpServersConfig"
	ExternalAgentDetectedConnectorSourceSessionToolUse         ExternalAgentDetectedConnectorSource = "sessionToolUse"
)

type ExternalAgentConfigImportCompletedNotification struct {
	ImportID        string                                `json:"importId"`
	ItemTypeResults []ExternalAgentConfigImportTypeResult `json:"itemTypeResults"`
}

type ExternalAgentConfigImportTypeResult struct {
	Failures  []ExternalAgentConfigImportItemTypeFailure `json:"failures"`
	ItemType  ExternalAgentConfigMigrationItemType       `json:"itemType"`
	Successes []ExternalAgentConfigImportItemTypeSuccess `json:"successes"`
}

type ExternalAgentConfigImportItemTypeSuccess struct {
	Cwd      *string                              `json:"cwd,omitempty"`
	ItemType ExternalAgentConfigMigrationItemType `json:"itemType"`
	Source   *string                              `json:"source,omitempty"`
	Target   *string                              `json:"target,omitempty"`
	Title    *string                              `json:"title,omitempty"`
}

type ExternalAgentConfigImportHistoriesReadResponse struct {
	Connectors []ExternalAgentImportedConnectorCandidate `json:"connectors"`
	Data       []ExternalAgentConfigImportHistory        `json:"data"`
}

type ExternalAgentImportedConnectorCandidate struct {
	Name         string                               `json:"name"`
	SessionCount int64                                `json:"sessionCount"`
	Source       ExternalAgentImportedConnectorSource `json:"source"`
}

type ExternalAgentImportedConnectorSource string

const (
	ExternalAgentImportedConnectorSourceRemoteMcpServersConfig ExternalAgentImportedConnectorSource = "remoteMcpServersConfig"
)

type ExternalAgentConfigImportHistory struct {
	CompletedAtMs int64                                      `json:"completedAtMs"`
	Failures      []ExternalAgentConfigImportItemTypeFailure `json:"failures"`
	ImportID      string                                     `json:"importId"`
	ProviderID    *string                                    `json:"providerId,omitempty"`
	Successes     []ExternalAgentConfigImportItemTypeSuccess `json:"successes"`
}

type ExternalAgentConfigImportHistoryRecordResponse struct {
	ImportID string `json:"importId"`
}

type ExternalAgentConfigImportProgressNotification struct {
	ImportID        string                                `json:"importId"`
	ItemTypeResults []ExternalAgentConfigImportTypeResult `json:"itemTypeResults"`
}

type ExternalAgentConfigImportResponse struct {
	ImportID string `json:"importId"`
}

type FeedbackUploadResponse struct {
	ThreadID string `json:"threadId"`
}

type FileChangeApprovalDecision struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FileChangeApprovalDecision) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "accept") {
		var v FileChangeApprovalDecisionVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "acceptForSession") {
		var v FileChangeApprovalDecisionVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "decline") {
		var v FileChangeApprovalDecisionVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "cancel") {
		var v FileChangeApprovalDecisionVariant4
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FileChangeApprovalDecision) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type FileChangeApprovalDecisionVariant1 string

const (
	FileChangeApprovalDecisionVariant1Accept FileChangeApprovalDecisionVariant1 = "accept"
)

type FileChangeApprovalDecisionVariant2 string

const (
	FileChangeApprovalDecisionVariant2AcceptForSession FileChangeApprovalDecisionVariant2 = "acceptForSession"
)

type FileChangeApprovalDecisionVariant3 string

const (
	FileChangeApprovalDecisionVariant3Decline FileChangeApprovalDecisionVariant3 = "decline"
)

type FileChangeApprovalDecisionVariant4 string

const (
	FileChangeApprovalDecisionVariant4Cancel FileChangeApprovalDecisionVariant4 = "cancel"
)

type FileChangeOutputDeltaNotification struct {
	Delta    string `json:"delta"`
	ItemID   string `json:"itemId"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type FileChangePatchUpdatedNotification struct {
	Changes  []FileUpdateChange `json:"changes"`
	ItemID   string             `json:"itemId"`
	ThreadID string             `json:"threadId"`
	TurnID   string             `json:"turnId"`
}

type FileUpdateChange struct {
	Diff string          `json:"diff"`
	Kind PatchChangeKind `json:"kind"`
	Path string          `json:"path"`
}

type PatchChangeKind struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *PatchChangeKind) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "add":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v AddPatchChangeKind
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "delete":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v DeletePatchChangeKind
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "update":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v UpdatePatchChangeKind
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v AddPatchChangeKind
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v DeletePatchChangeKind
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v UpdatePatchChangeKind
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u PatchChangeKind) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type AddPatchChangeKind struct {
	Type string `json:"type"`
}

type DeletePatchChangeKind struct {
	Type string `json:"type"`
}

type UpdatePatchChangeKind struct {
	MovePath *string `json:"move_path,omitempty"`
	Type     string  `json:"type"`
}

type FileChangeRequestApprovalParams struct {
	GrantRoot   *string `json:"grantRoot,omitempty"`
	ItemID      string  `json:"itemId"`
	Reason      *string `json:"reason,omitempty"`
	StartedAtMs int64   `json:"startedAtMs"`
	ThreadID    string  `json:"threadId"`
	TurnID      string  `json:"turnId"`
}

type FileChangeRequestApprovalResponse struct {
	Decision FileChangeApprovalDecision `json:"decision"`
}

type FsChangedNotification struct {
	ChangedPaths []AbsolutePathBuf `json:"changedPaths"`
	WatchID      string            `json:"watchId"`
}

type FsCopyResponse struct {
}

type FsCreateDirectoryResponse struct {
}

type FsGetMetadataResponse struct {
	CreatedAtMs  int64 `json:"createdAtMs"`
	IsDirectory  bool  `json:"isDirectory"`
	IsFile       bool  `json:"isFile"`
	IsSymlink    bool  `json:"isSymlink"`
	ModifiedAtMs int64 `json:"modifiedAtMs"`
}

type FsReadDirectoryEntry struct {
	FileName    string `json:"fileName"`
	IsDirectory bool   `json:"isDirectory"`
	IsFile      bool   `json:"isFile"`
}

type FsReadDirectoryResponse struct {
	Entries []FsReadDirectoryEntry `json:"entries"`
}

type FsReadFileResponse struct {
	DataBase64 string `json:"dataBase64"`
}

type FsRemoveResponse struct {
}

type FsUnwatchResponse struct {
}

type FsWatchResponse struct {
	Path AbsolutePathBuf `json:"path"`
}

type FsWriteFileResponse struct {
}

type FunctionCallOutputBody struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FunctionCallOutputBody) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksString(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksArray(data) {
		var v UnknownUnionValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FunctionCallOutputBody) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type FunctionCallOutputContentItem struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *FunctionCallOutputContentItem) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "input_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v InputTextFunctionCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "input_image":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "image_url", "type")) {
					break
				}
				var v InputImageFunctionCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "input_audio":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "audio_url", "type")) {
					break
				}
				var v InputAudioFunctionCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "encrypted_content":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type")) {
					break
				}
				var v EncryptedContentFunctionCallOutputContentItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v InputTextFunctionCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "image_url", "type") {
		var v InputImageFunctionCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "audio_url", "type") {
		var v InputAudioFunctionCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type") {
		var v EncryptedContentFunctionCallOutputContentItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u FunctionCallOutputContentItem) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InputTextFunctionCallOutputContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type InputImageFunctionCallOutputContentItem struct {
	Detail   *ImageDetail `json:"detail,omitempty"`
	ImageUrl string       `json:"image_url"`
	Type     string       `json:"type"`
}

type InputAudioFunctionCallOutputContentItem struct {
	AudioUrl string `json:"audio_url"`
	Type     string `json:"type"`
}

type EncryptedContentFunctionCallOutputContentItem struct {
	EncryptedContent string `json:"encrypted_content"`
	Type             string `json:"type"`
}

type FuzzyFileSearchMatchType string

const (
	FuzzyFileSearchMatchTypeFile      FuzzyFileSearchMatchType = "file"
	FuzzyFileSearchMatchTypeDirectory FuzzyFileSearchMatchType = "directory"
)

type FuzzyFileSearchResponse struct {
	Files []FuzzyFileSearchResult `json:"files"`
}

type FuzzyFileSearchResult struct {
	FileName  string                   `json:"file_name"`
	Indices   []int64                  `json:"indices,omitempty"`
	MatchType FuzzyFileSearchMatchType `json:"match_type"`
	Path      string                   `json:"path"`
	Root      string                   `json:"root"`
	Score     int64                    `json:"score"`
}

type FuzzyFileSearchSessionCompletedNotification struct {
	SessionID string `json:"sessionId"`
}

type FuzzyFileSearchSessionUpdatedNotification struct {
	Files     []FuzzyFileSearchResult `json:"files"`
	Query     string                  `json:"query"`
	SessionID string                  `json:"sessionId"`
}

type GetAccountRateLimitsResponse struct {
	RateLimitResetCredits *RateLimitResetCreditsSummary `json:"rateLimitResetCredits,omitempty"`
	RateLimits            RateLimitSnapshot             `json:"rateLimits"`
	RateLimitsByLimitID   map[string]RateLimitSnapshot  `json:"rateLimitsByLimitId,omitempty"`
}

type RateLimitResetCreditsSummary struct {
	AvailableCount int64                  `json:"availableCount"`
	Credits        []RateLimitResetCredit `json:"credits,omitempty"`
}

type RateLimitResetCredit struct {
	Description *string                    `json:"description,omitempty"`
	ExpiresAt   *int64                     `json:"expiresAt,omitempty"`
	GrantedAt   int64                      `json:"grantedAt"`
	ID          string                     `json:"id"`
	ResetType   RateLimitResetType         `json:"resetType"`
	Status      RateLimitResetCreditStatus `json:"status"`
	Title       *string                    `json:"title,omitempty"`
}

type RateLimitResetType string

const (
	RateLimitResetTypeCodexRateLimits RateLimitResetType = "codexRateLimits"
	RateLimitResetTypeUnknown         RateLimitResetType = "unknown"
)

type RateLimitResetCreditStatus string

const (
	RateLimitResetCreditStatusAvailable RateLimitResetCreditStatus = "available"
	RateLimitResetCreditStatusRedeeming RateLimitResetCreditStatus = "redeeming"
	RateLimitResetCreditStatusRedeemed  RateLimitResetCreditStatus = "redeemed"
	RateLimitResetCreditStatusUnknown   RateLimitResetCreditStatus = "unknown"
)

type GetAccountResponse struct {
	Account            *Account `json:"account,omitempty"`
	RequiresOpenaiAuth bool     `json:"requiresOpenaiAuth"`
}

type GetAccountTokenUsageResponse struct {
	DailyUsageBuckets []AccountTokenUsageDailyBucket `json:"dailyUsageBuckets,omitempty"`
	Summary           AccountTokenUsageSummary       `json:"summary"`
}

type GetWorkspaceMessagesResponse struct {
	FeatureEnabled bool               `json:"featureEnabled"`
	Messages       []WorkspaceMessage `json:"messages"`
}

type WorkspaceMessage struct {
	ArchivedAt  *int64               `json:"archivedAt,omitempty"`
	CreatedAt   *int64               `json:"createdAt,omitempty"`
	MessageBody string               `json:"messageBody"`
	MessageID   string               `json:"messageId"`
	MessageType WorkspaceMessageType `json:"messageType"`
}

type WorkspaceMessageType string

const (
	WorkspaceMessageTypeHeadline     WorkspaceMessageType = "headline"
	WorkspaceMessageTypeAnnouncement WorkspaceMessageType = "announcement"
	WorkspaceMessageTypeUnknown      WorkspaceMessageType = "unknown"
)

type GitInfo struct {
	Branch    *string `json:"branch,omitempty"`
	OriginUrl *string `json:"originUrl,omitempty"`
	Sha       *string `json:"sha,omitempty"`
}

type GrantedPermissionProfile struct {
	FileSystem *AdditionalFileSystemPermissions `json:"fileSystem,omitempty"`
	Network    *AdditionalNetworkPermissions    `json:"network,omitempty"`
}

type GuardianApprovalReview struct {
	Rationale         *string                      `json:"rationale,omitempty"`
	RiskLevel         *GuardianRiskLevel           `json:"riskLevel,omitempty"`
	Status            GuardianApprovalReviewStatus `json:"status"`
	UserAuthorization *GuardianUserAuthorization   `json:"userAuthorization,omitempty"`
}

type GuardianRiskLevel string

const (
	GuardianRiskLevelLow      GuardianRiskLevel = "low"
	GuardianRiskLevelMedium   GuardianRiskLevel = "medium"
	GuardianRiskLevelHigh     GuardianRiskLevel = "high"
	GuardianRiskLevelCritical GuardianRiskLevel = "critical"
)

type GuardianApprovalReviewStatus string

const (
	GuardianApprovalReviewStatusInProgress GuardianApprovalReviewStatus = "inProgress"
	GuardianApprovalReviewStatusApproved   GuardianApprovalReviewStatus = "approved"
	GuardianApprovalReviewStatusDenied     GuardianApprovalReviewStatus = "denied"
	GuardianApprovalReviewStatusTimedOut   GuardianApprovalReviewStatus = "timedOut"
	GuardianApprovalReviewStatusAborted    GuardianApprovalReviewStatus = "aborted"
)

type GuardianUserAuthorization string

const (
	GuardianUserAuthorizationUnknown GuardianUserAuthorization = "unknown"
	GuardianUserAuthorizationLow     GuardianUserAuthorization = "low"
	GuardianUserAuthorizationMedium  GuardianUserAuthorization = "medium"
	GuardianUserAuthorizationHigh    GuardianUserAuthorization = "high"
)

type GuardianApprovalReviewAction struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *GuardianApprovalReviewAction) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "command":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "cwd", "source", "type")) {
					break
				}
				var v CommandGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "execve":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "argv", "cwd", "program", "source", "type")) {
					break
				}
				var v ExecveGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "applyPatch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "cwd", "files", "type")) {
					break
				}
				var v ApplyPatchGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "networkAccess":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "host", "port", "protocol", "target", "type")) {
					break
				}
				var v NetworkAccessGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpToolCall":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "server", "toolName", "type")) {
					break
				}
				var v McpToolCallGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "requestPermissions":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "permissions", "type")) {
					break
				}
				var v RequestPermissionsGuardianApprovalReviewAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "cwd", "source", "type") {
		var v CommandGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "argv", "cwd", "program", "source", "type") {
		var v ExecveGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "cwd", "files", "type") {
		var v ApplyPatchGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "host", "port", "protocol", "target", "type") {
		var v NetworkAccessGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "server", "toolName", "type") {
		var v McpToolCallGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "permissions", "type") {
		var v RequestPermissionsGuardianApprovalReviewAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u GuardianApprovalReviewAction) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type CommandGuardianApprovalReviewAction struct {
	Command string                `json:"command"`
	Cwd     AbsolutePathBuf       `json:"cwd"`
	Source  GuardianCommandSource `json:"source"`
	Type    string                `json:"type"`
}

type GuardianCommandSource string

const (
	GuardianCommandSourceShell       GuardianCommandSource = "shell"
	GuardianCommandSourceUnifiedExec GuardianCommandSource = "unifiedExec"
)

type ExecveGuardianApprovalReviewAction struct {
	Argv    []string              `json:"argv"`
	Cwd     AbsolutePathBuf       `json:"cwd"`
	Program string                `json:"program"`
	Source  GuardianCommandSource `json:"source"`
	Type    string                `json:"type"`
}

type ApplyPatchGuardianApprovalReviewAction struct {
	Cwd   AbsolutePathBuf   `json:"cwd"`
	Files []AbsolutePathBuf `json:"files"`
	Type  string            `json:"type"`
}

type NetworkAccessGuardianApprovalReviewAction struct {
	Host     string                  `json:"host"`
	Port     int64                   `json:"port"`
	Protocol NetworkApprovalProtocol `json:"protocol"`
	Target   string                  `json:"target"`
	Type     string                  `json:"type"`
}

type McpToolCallGuardianApprovalReviewAction struct {
	ConnectorID   *string `json:"connectorId,omitempty"`
	ConnectorName *string `json:"connectorName,omitempty"`
	Server        string  `json:"server"`
	ToolName      string  `json:"toolName"`
	ToolTitle     *string `json:"toolTitle,omitempty"`
	Type          string  `json:"type"`
}

type RequestPermissionsGuardianApprovalReviewAction struct {
	Permissions RequestPermissionProfile `json:"permissions"`
	Reason      *string                  `json:"reason,omitempty"`
	Type        string                   `json:"type"`
}

type RequestPermissionProfile struct {
	FileSystem *AdditionalFileSystemPermissions `json:"fileSystem,omitempty"`
	Network    *AdditionalNetworkPermissions    `json:"network,omitempty"`
}

type GuardianWarningNotification struct {
	Message  string `json:"message"`
	ThreadID string `json:"threadId"`
}

type HookCompletedNotification struct {
	Run      HookRunSummary `json:"run"`
	ThreadID string         `json:"threadId"`
	TurnID   *string        `json:"turnId,omitempty"`
}

type HookRunSummary struct {
	CompletedAt   *int64            `json:"completedAt,omitempty"`
	DisplayOrder  int64             `json:"displayOrder"`
	DurationMs    *int64            `json:"durationMs,omitempty"`
	Entries       []HookOutputEntry `json:"entries"`
	EventName     HookEventName     `json:"eventName"`
	ExecutionMode HookExecutionMode `json:"executionMode"`
	HandlerType   HookHandlerType   `json:"handlerType"`
	ID            string            `json:"id"`
	Scope         HookScope         `json:"scope"`
	Source        *HookSource       `json:"source,omitempty"`
	SourcePath    AbsolutePathBuf   `json:"sourcePath"`
	StartedAt     int64             `json:"startedAt"`
	Status        HookRunStatus     `json:"status"`
	StatusMessage *string           `json:"statusMessage,omitempty"`
}

type HookOutputEntry struct {
	Kind HookOutputEntryKind `json:"kind"`
	Text string              `json:"text"`
}

type HookOutputEntryKind string

const (
	HookOutputEntryKindWarning  HookOutputEntryKind = "warning"
	HookOutputEntryKindStop     HookOutputEntryKind = "stop"
	HookOutputEntryKindFeedback HookOutputEntryKind = "feedback"
	HookOutputEntryKindContext  HookOutputEntryKind = "context"
	HookOutputEntryKindError    HookOutputEntryKind = "error"
)

type HookEventName string

const (
	HookEventNamePreToolUse        HookEventName = "preToolUse"
	HookEventNamePermissionRequest HookEventName = "permissionRequest"
	HookEventNamePostToolUse       HookEventName = "postToolUse"
	HookEventNamePreCompact        HookEventName = "preCompact"
	HookEventNamePostCompact       HookEventName = "postCompact"
	HookEventNameSessionStart      HookEventName = "sessionStart"
	HookEventNameSessionEnd        HookEventName = "sessionEnd"
	HookEventNameUserPromptSubmit  HookEventName = "userPromptSubmit"
	HookEventNameSubagentStart     HookEventName = "subagentStart"
	HookEventNameSubagentStop      HookEventName = "subagentStop"
	HookEventNameStop              HookEventName = "stop"
)

type HookExecutionMode string

const (
	HookExecutionModeSync  HookExecutionMode = "sync"
	HookExecutionModeAsync HookExecutionMode = "async"
)

type HookHandlerType string

const (
	HookHandlerTypeCommand HookHandlerType = "command"
	HookHandlerTypePrompt  HookHandlerType = "prompt"
	HookHandlerTypeAgent   HookHandlerType = "agent"
)

type HookScope string

const (
	HookScopeThread HookScope = "thread"
	HookScopeTurn   HookScope = "turn"
)

type HookSource string

const (
	HookSourceSystem                  HookSource = "system"
	HookSourceUser                    HookSource = "user"
	HookSourceProject                 HookSource = "project"
	HookSourceMdm                     HookSource = "mdm"
	HookSourceSessionFlags            HookSource = "sessionFlags"
	HookSourcePlugin                  HookSource = "plugin"
	HookSourceCloudRequirements       HookSource = "cloudRequirements"
	HookSourceCloudManagedConfig      HookSource = "cloudManagedConfig"
	HookSourceLegacyManagedConfigFile HookSource = "legacyManagedConfigFile"
	HookSourceLegacyManagedConfigMdm  HookSource = "legacyManagedConfigMdm"
	HookSourceUnknown                 HookSource = "unknown"
)

type HookRunStatus string

const (
	HookRunStatusRunning   HookRunStatus = "running"
	HookRunStatusCompleted HookRunStatus = "completed"
	HookRunStatusFailed    HookRunStatus = "failed"
	HookRunStatusBlocked   HookRunStatus = "blocked"
	HookRunStatusStopped   HookRunStatus = "stopped"
)

type HookErrorInfo struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

type HookMetadata struct {
	AdditionalContextLimit *int64          `json:"additionalContextLimit,omitempty"`
	Command                *string         `json:"command,omitempty"`
	CurrentHash            string          `json:"currentHash"`
	DisplayOrder           int64           `json:"displayOrder"`
	Enabled                bool            `json:"enabled"`
	EventName              HookEventName   `json:"eventName"`
	HandlerType            HookHandlerType `json:"handlerType"`
	IsManaged              bool            `json:"isManaged"`
	Key                    string          `json:"key"`
	Matcher                *string         `json:"matcher,omitempty"`
	PluginID               *string         `json:"pluginId,omitempty"`
	Source                 HookSource      `json:"source"`
	SourcePath             AbsolutePathBuf `json:"sourcePath"`
	StatusMessage          *string         `json:"statusMessage,omitempty"`
	TimeoutSec             int64           `json:"timeoutSec"`
	TrustStatus            HookTrustStatus `json:"trustStatus"`
}

type HookTrustStatus string

const (
	HookTrustStatusManaged   HookTrustStatus = "managed"
	HookTrustStatusUntrusted HookTrustStatus = "untrusted"
	HookTrustStatusTrusted   HookTrustStatus = "trusted"
	HookTrustStatusModified  HookTrustStatus = "modified"
)

type HookPromptFragment struct {
	HookRunID string `json:"hookRunId"`
	Text      string `json:"text"`
}

type HookStartedNotification struct {
	Run      HookRunSummary `json:"run"`
	ThreadID string         `json:"threadId"`
	TurnID   *string        `json:"turnId,omitempty"`
}

type HooksListEntry struct {
	Cwd      string          `json:"cwd"`
	Errors   []HookErrorInfo `json:"errors"`
	Hooks    []HookMetadata  `json:"hooks"`
	Warnings []string        `json:"warnings"`
}

type HooksListResponse struct {
	Data []HooksListEntry `json:"data"`
}

type InitializeResponse struct {
	CodexHome      AbsolutePathBuf `json:"codexHome"`
	PlatformFamily string          `json:"platformFamily"`
	PlatformOs     string          `json:"platformOs"`
	UserAgent      string          `json:"userAgent"`
}

type InputModality struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *InputModality) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "text") {
		var v InputModalityVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "image") {
		var v InputModalityVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "audio") {
		var v InputModalityVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u InputModality) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type InputModalityVariant1 string

const (
	InputModalityVariant1Text InputModalityVariant1 = "text"
)

type InputModalityVariant2 string

const (
	InputModalityVariant2Image InputModalityVariant2 = "image"
)

type InputModalityVariant3 string

const (
	InputModalityVariant3Audio InputModalityVariant3 = "audio"
)

type InternalChatMessageMetadataPassthrough struct {
	TurnID *string `json:"turn_id,omitempty"`
}

type ItemCompletedNotification struct {
	CompletedAtMs int64      `json:"completedAtMs"`
	Item          ThreadItem `json:"item"`
	ThreadID      string     `json:"threadId"`
	TurnID        string     `json:"turnId"`
}

type ThreadItem struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ThreadItem) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "userMessage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "content", "id", "type")) {
					break
				}
				var v UserMessageThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "hookPrompt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "fragments", "id", "type")) {
					break
				}
				var v HookPromptThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "agentMessage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "text", "type")) {
					break
				}
				var v AgentMessageThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "plan":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "text", "type")) {
					break
				}
				var v PlanThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "reasoning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "type")) {
					break
				}
				var v ReasoningThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "commandExecution":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "commandActions", "cwd", "id", "status", "type")) {
					break
				}
				var v CommandExecutionThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fileChange":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "changes", "id", "status", "type")) {
					break
				}
				var v FileChangeThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpToolCall":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "arguments", "id", "server", "status", "tool", "type")) {
					break
				}
				var v McpToolCallThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "dynamicToolCall":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "arguments", "id", "status", "tool", "type")) {
					break
				}
				var v DynamicToolCallThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "collabAgentToolCall":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "agentsStates", "id", "receiverThreadIds", "senderThreadId", "status", "tool", "type")) {
					break
				}
				var v CollabAgentToolCallThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "subAgentActivity":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "agentPath", "agentThreadId", "id", "kind", "type")) {
					break
				}
				var v SubAgentActivityThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "webSearch":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "query", "type")) {
					break
				}
				var v WebSearchThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "imageView":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "path", "type")) {
					break
				}
				var v ImageViewThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "sleep":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "durationMs", "id", "type")) {
					break
				}
				var v SleepThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "imageGeneration":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "result", "status", "type")) {
					break
				}
				var v ImageGenerationThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "enteredReviewMode":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "review", "type")) {
					break
				}
				var v EnteredReviewModeThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "exitedReviewMode":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "review", "type")) {
					break
				}
				var v ExitedReviewModeThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "contextCompaction":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "type")) {
					break
				}
				var v ContextCompactionThreadItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "content", "id", "type") {
		var v UserMessageThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "fragments", "id", "type") {
		var v HookPromptThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "text", "type") {
		var v AgentMessageThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "text", "type") {
		var v PlanThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "type") {
		var v ReasoningThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "commandActions", "cwd", "id", "status", "type") {
		var v CommandExecutionThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "changes", "id", "status", "type") {
		var v FileChangeThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "arguments", "id", "server", "status", "tool", "type") {
		var v McpToolCallThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "arguments", "id", "status", "tool", "type") {
		var v DynamicToolCallThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "agentsStates", "id", "receiverThreadIds", "senderThreadId", "status", "tool", "type") {
		var v CollabAgentToolCallThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "agentPath", "agentThreadId", "id", "kind", "type") {
		var v SubAgentActivityThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "query", "type") {
		var v WebSearchThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "path", "type") {
		var v ImageViewThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "durationMs", "id", "type") {
		var v SleepThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "result", "status", "type") {
		var v ImageGenerationThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "review", "type") {
		var v EnteredReviewModeThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "review", "type") {
		var v ExitedReviewModeThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "type") {
		var v ContextCompactionThreadItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ThreadItem) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type UserMessageThreadItem struct {
	ClientID *string     `json:"clientId,omitempty"`
	Content  []UserInput `json:"content"`
	ID       string      `json:"id"`
	Type     string      `json:"type"`
}

type HookPromptThreadItem struct {
	Fragments []HookPromptFragment `json:"fragments"`
	ID        string               `json:"id"`
	Type      string               `json:"type"`
}

type AgentMessageThreadItem struct {
	ID             string          `json:"id"`
	MemoryCitation *MemoryCitation `json:"memoryCitation,omitempty"`
	Phase          *MessagePhase   `json:"phase,omitempty"`
	Text           string          `json:"text"`
	Type           string          `json:"type"`
}

type MemoryCitation struct {
	Entries   []MemoryCitationEntry `json:"entries"`
	ThreadIds []string              `json:"threadIds"`
}

type MemoryCitationEntry struct {
	LineEnd   int64  `json:"lineEnd"`
	LineStart int64  `json:"lineStart"`
	Note      string `json:"note"`
	Path      string `json:"path"`
}

type MessagePhase struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *MessagePhase) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "commentary") {
		var v MessagePhaseVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "final_answer") {
		var v MessagePhaseVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u MessagePhase) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type MessagePhaseVariant1 string

const (
	MessagePhaseVariant1Commentary MessagePhaseVariant1 = "commentary"
)

type MessagePhaseVariant2 string

const (
	MessagePhaseVariant2FinalAnswer MessagePhaseVariant2 = "final_answer"
)

type PlanThreadItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type ReasoningThreadItem struct {
	Content []string `json:"content,omitempty"`
	ID      string   `json:"id"`
	Summary []string `json:"summary,omitempty"`
	Type    string   `json:"type"`
}

type CommandExecutionThreadItem struct {
	AggregatedOutput *string                 `json:"aggregatedOutput,omitempty"`
	Command          string                  `json:"command"`
	CommandActions   []CommandAction         `json:"commandActions"`
	Cwd              LegacyAppPathString     `json:"cwd"`
	DurationMs       *int64                  `json:"durationMs,omitempty"`
	ExitCode         *int64                  `json:"exitCode,omitempty"`
	ID               string                  `json:"id"`
	PluginID         *string                 `json:"pluginId,omitempty"`
	ProcessID        *string                 `json:"processId,omitempty"`
	ScriptPath       *string                 `json:"scriptPath,omitempty"`
	Source           *CommandExecutionSource `json:"source,omitempty"`
	Status           CommandExecutionStatus  `json:"status"`
	Type             string                  `json:"type"`
}

type FileChangeThreadItem struct {
	Changes []FileUpdateChange `json:"changes"`
	ID      string             `json:"id"`
	Status  PatchApplyStatus   `json:"status"`
	Type    string             `json:"type"`
}

type PatchApplyStatus string

const (
	PatchApplyStatusInProgress PatchApplyStatus = "inProgress"
	PatchApplyStatusCompleted  PatchApplyStatus = "completed"
	PatchApplyStatusFailed     PatchApplyStatus = "failed"
	PatchApplyStatusDeclined   PatchApplyStatus = "declined"
)

type McpToolCallThreadItem struct {
	AppContext        *McpToolCallAppContext `json:"appContext,omitempty"`
	Arguments         any                    `json:"arguments"`
	DurationMs        *int64                 `json:"durationMs,omitempty"`
	Error             *McpToolCallError      `json:"error,omitempty"`
	ID                string                 `json:"id"`
	McpAppResourceUri *string                `json:"mcpAppResourceUri,omitempty"`
	PluginID          *string                `json:"pluginId,omitempty"`
	ReadOnlyHint      *bool                  `json:"readOnlyHint,omitempty"`
	Result            *McpToolCallResult     `json:"result,omitempty"`
	Server            string                 `json:"server"`
	Status            McpToolCallStatus      `json:"status"`
	Tool              string                 `json:"tool"`
	Type              string                 `json:"type"`
}

type McpToolCallAppContext struct {
	ActionName  *string `json:"actionName,omitempty"`
	AppName     *string `json:"appName,omitempty"`
	ConnectorID string  `json:"connectorId"`
	LinkID      *string `json:"linkId,omitempty"`
	ResourceUri *string `json:"resourceUri,omitempty"`
}

type McpToolCallError struct {
	Message string `json:"message"`
}

type McpToolCallResult struct {
	Meta              any   `json:"_meta,omitempty"`
	Content           []any `json:"content"`
	StructuredContent any   `json:"structuredContent,omitempty"`
}

type McpToolCallStatus string

const (
	McpToolCallStatusInProgress McpToolCallStatus = "inProgress"
	McpToolCallStatusCompleted  McpToolCallStatus = "completed"
	McpToolCallStatusFailed     McpToolCallStatus = "failed"
)

type DynamicToolCallThreadItem struct {
	Arguments    any                                `json:"arguments"`
	ContentItems []DynamicToolCallOutputContentItem `json:"contentItems,omitempty"`
	DurationMs   *int64                             `json:"durationMs,omitempty"`
	ID           string                             `json:"id"`
	Namespace    *string                            `json:"namespace,omitempty"`
	Status       DynamicToolCallStatus              `json:"status"`
	Success      *bool                              `json:"success,omitempty"`
	Tool         string                             `json:"tool"`
	Type         string                             `json:"type"`
}

type CollabAgentToolCallThreadItem struct {
	AgentsStates      map[string]CollabAgentState `json:"agentsStates"`
	ID                string                      `json:"id"`
	Model             *string                     `json:"model,omitempty"`
	Prompt            *string                     `json:"prompt,omitempty"`
	ReasoningEffort   *ReasoningEffort            `json:"reasoningEffort,omitempty"`
	ReceiverThreadIds []string                    `json:"receiverThreadIds"`
	SenderThreadID    string                      `json:"senderThreadId"`
	Status            CollabAgentToolCallStatus   `json:"status"`
	Tool              CollabAgentTool             `json:"tool"`
	Type              string                      `json:"type"`
}

type SubAgentActivityThreadItem struct {
	AgentPath     string               `json:"agentPath"`
	AgentThreadID string               `json:"agentThreadId"`
	ID            string               `json:"id"`
	Kind          SubAgentActivityKind `json:"kind"`
	Type          string               `json:"type"`
}

type SubAgentActivityKind string

const (
	SubAgentActivityKindStarted     SubAgentActivityKind = "started"
	SubAgentActivityKindInteracted  SubAgentActivityKind = "interacted"
	SubAgentActivityKindInterrupted SubAgentActivityKind = "interrupted"
)

type WebSearchThreadItem struct {
	Action  *WebSearchAction `json:"action,omitempty"`
	ID      string           `json:"id"`
	Query   string           `json:"query"`
	Results []any            `json:"results,omitempty"`
	Type    string           `json:"type"`
}

type WebSearchAction struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *WebSearchAction) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "search":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v SearchWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "openPage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v OpenPageWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "findInPage":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v FindInPageWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "other":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v OtherWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v SearchWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v OpenPageWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v FindInPageWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v OtherWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u WebSearchAction) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SearchWebSearchAction struct {
	Queries []string `json:"queries,omitempty"`
	Query   *string  `json:"query,omitempty"`
	Type    string   `json:"type"`
}

type OpenPageWebSearchAction struct {
	Type string  `json:"type"`
	Url  *string `json:"url,omitempty"`
}

type FindInPageWebSearchAction struct {
	Pattern *string `json:"pattern,omitempty"`
	Type    string  `json:"type"`
	Url     *string `json:"url,omitempty"`
}

type OtherWebSearchAction struct {
	Type string `json:"type"`
}

type ImageViewThreadItem struct {
	ID   string              `json:"id"`
	Path LegacyAppPathString `json:"path"`
	Type string              `json:"type"`
}

type SleepThreadItem struct {
	DurationMs int64  `json:"durationMs"`
	ID         string `json:"id"`
	Type       string `json:"type"`
}

type ImageGenerationThreadItem struct {
	ID            string           `json:"id"`
	Result        string           `json:"result"`
	RevisedPrompt *string          `json:"revisedPrompt,omitempty"`
	SavedPath     *AbsolutePathBuf `json:"savedPath,omitempty"`
	Status        string           `json:"status"`
	Type          string           `json:"type"`
}

type EnteredReviewModeThreadItem struct {
	ID     string `json:"id"`
	Review string `json:"review"`
	Type   string `json:"type"`
}

type ExitedReviewModeThreadItem struct {
	ID     string `json:"id"`
	Review string `json:"review"`
	Type   string `json:"type"`
}

type ContextCompactionThreadItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type ItemGuardianApprovalReviewCompletedNotification struct {
	Action         GuardianApprovalReviewAction `json:"action"`
	CompletedAtMs  int64                        `json:"completedAtMs"`
	DecisionSource AutoReviewDecisionSource     `json:"decisionSource"`
	Review         GuardianApprovalReview       `json:"review"`
	ReviewID       string                       `json:"reviewId"`
	StartedAtMs    int64                        `json:"startedAtMs"`
	TargetItemID   *string                      `json:"targetItemId,omitempty"`
	ThreadID       string                       `json:"threadId"`
	TurnID         string                       `json:"turnId"`
}

type ItemGuardianApprovalReviewStartedNotification struct {
	Action       GuardianApprovalReviewAction `json:"action"`
	Review       GuardianApprovalReview       `json:"review"`
	ReviewID     string                       `json:"reviewId"`
	StartedAtMs  int64                        `json:"startedAtMs"`
	TargetItemID *string                      `json:"targetItemId,omitempty"`
	ThreadID     string                       `json:"threadId"`
	TurnID       string                       `json:"turnId"`
}

type ItemStartedNotification struct {
	Item        ThreadItem `json:"item"`
	StartedAtMs int64      `json:"startedAtMs"`
	ThreadID    string     `json:"threadId"`
	TurnID      string     `json:"turnId"`
}

type JSONRPCError struct {
	Error JSONRPCErrorError `json:"error"`
	ID    RequestId         `json:"id"`
}

type JSONRPCErrorError struct {
	Code    int64  `json:"code"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
}

type JSONRPCMessage struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *JSONRPCMessage) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u JSONRPCMessage) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type JSONRPCNotification struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type JSONRPCRequest struct {
	ID     RequestId        `json:"id"`
	Method string           `json:"method"`
	Params any              `json:"params,omitempty"`
	Trace  *W3cTraceContext `json:"trace,omitempty"`
}

type W3cTraceContext struct {
	Traceparent *string `json:"traceparent,omitempty"`
	Tracestate  *string `json:"tracestate,omitempty"`
}

type JSONRPCResponse struct {
	ID     RequestId `json:"id"`
	Result any       `json:"result"`
}

type ListMcpServerStatusResponse struct {
	Data       []McpServerStatus `json:"data"`
	NextCursor *string           `json:"nextCursor,omitempty"`
}

type McpServerStatus struct {
	AuthStatus        McpAuthStatus      `json:"authStatus"`
	Name              string             `json:"name"`
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	Resources         []Resource         `json:"resources"`
	ServerInfo        *McpServerInfo     `json:"serverInfo,omitempty"`
	Tools             map[string]Tool    `json:"tools"`
}

type McpAuthStatus string

const (
	McpAuthStatusUnknown     McpAuthStatus = "unknown"
	McpAuthStatusUnsupported McpAuthStatus = "unsupported"
	McpAuthStatusNotLoggedIn McpAuthStatus = "notLoggedIn"
	McpAuthStatusBearerToken McpAuthStatus = "bearerToken"
	McpAuthStatusOAuth       McpAuthStatus = "oAuth"
)

type ResourceTemplate struct {
	Annotations any     `json:"annotations,omitempty"`
	Description *string `json:"description,omitempty"`
	MimeType    *string `json:"mimeType,omitempty"`
	Name        string  `json:"name"`
	Title       *string `json:"title,omitempty"`
	UriTemplate string  `json:"uriTemplate"`
}

type Resource struct {
	Meta        any     `json:"_meta,omitempty"`
	Annotations any     `json:"annotations,omitempty"`
	Description *string `json:"description,omitempty"`
	Icons       []any   `json:"icons,omitempty"`
	MimeType    *string `json:"mimeType,omitempty"`
	Name        string  `json:"name"`
	Size        *int64  `json:"size,omitempty"`
	Title       *string `json:"title,omitempty"`
	Uri         string  `json:"uri"`
}

type McpServerInfo struct {
	Description *string `json:"description,omitempty"`
	Icons       []any   `json:"icons,omitempty"`
	Name        string  `json:"name"`
	Title       *string `json:"title,omitempty"`
	Version     string  `json:"version"`
	WebsiteUrl  *string `json:"websiteUrl,omitempty"`
}

type Tool struct {
	Meta         any     `json:"_meta,omitempty"`
	Annotations  any     `json:"annotations,omitempty"`
	Description  *string `json:"description,omitempty"`
	Icons        []any   `json:"icons,omitempty"`
	InputSchema  any     `json:"inputSchema"`
	Name         string  `json:"name"`
	OutputSchema any     `json:"outputSchema,omitempty"`
	Title        *string `json:"title,omitempty"`
}

type LocalShellAction struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *LocalShellAction) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "exec":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "command", "type")) {
					break
				}
				var v ExecLocalShellAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "command", "type") {
		var v ExecLocalShellAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u LocalShellAction) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ExecLocalShellAction struct {
	Command          []string          `json:"command"`
	Env              map[string]string `json:"env,omitempty"`
	TimeoutMs        *int64            `json:"timeout_ms,omitempty"`
	Type             string            `json:"type"`
	User             *string           `json:"user,omitempty"`
	WorkingDirectory *string           `json:"working_directory,omitempty"`
}

type LocalShellStatus string

const (
	LocalShellStatusCompleted  LocalShellStatus = "completed"
	LocalShellStatusInProgress LocalShellStatus = "in_progress"
	LocalShellStatusIncomplete LocalShellStatus = "incomplete"
)

type LoginAccountResponse struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *LoginAccountResponse) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "apiKey":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ApiKeyv2LoginAccountResponse
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgpt":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "authUrl", "loginId", "type")) {
					break
				}
				var v Chatgptv2LoginAccountResponse
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgptDeviceCode":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "loginId", "type", "userCode", "verificationUrl")) {
					break
				}
				var v ChatgptDeviceCodev2LoginAccountResponse
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "chatgptAuthTokens":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ChatgptAuthTokensv2LoginAccountResponse
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "amazonBedrock":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v AmazonBedrockv2LoginAccountResponse
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ApiKeyv2LoginAccountResponse
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "authUrl", "loginId", "type") {
		var v Chatgptv2LoginAccountResponse
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "loginId", "type", "userCode", "verificationUrl") {
		var v ChatgptDeviceCodev2LoginAccountResponse
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ChatgptAuthTokensv2LoginAccountResponse
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v AmazonBedrockv2LoginAccountResponse
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u LoginAccountResponse) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ApiKeyv2LoginAccountResponse struct {
	Type string `json:"type"`
}

type Chatgptv2LoginAccountResponse struct {
	AuthUrl string `json:"authUrl"`
	LoginID string `json:"loginId"`
	Type    string `json:"type"`
}

type ChatgptDeviceCodev2LoginAccountResponse struct {
	LoginID         string `json:"loginId"`
	Type            string `json:"type"`
	UserCode        string `json:"userCode"`
	VerificationUrl string `json:"verificationUrl"`
}

type ChatgptAuthTokensv2LoginAccountResponse struct {
	Type string `json:"type"`
}

type AmazonBedrockv2LoginAccountResponse struct {
	Type string `json:"type"`
}

type LogoutAccountResponse struct {
}

type ManagedHooksRequirements struct {
	PermissionRequest []ConfiguredHookMatcherGroup `json:"PermissionRequest"`
	PostCompact       []ConfiguredHookMatcherGroup `json:"PostCompact"`
	PostToolUse       []ConfiguredHookMatcherGroup `json:"PostToolUse"`
	PreCompact        []ConfiguredHookMatcherGroup `json:"PreCompact"`
	PreToolUse        []ConfiguredHookMatcherGroup `json:"PreToolUse"`
	SessionEnd        []ConfiguredHookMatcherGroup `json:"SessionEnd,omitempty"`
	SessionStart      []ConfiguredHookMatcherGroup `json:"SessionStart"`
	Stop              []ConfiguredHookMatcherGroup `json:"Stop"`
	SubagentStart     []ConfiguredHookMatcherGroup `json:"SubagentStart"`
	SubagentStop      []ConfiguredHookMatcherGroup `json:"SubagentStop"`
	UserPromptSubmit  []ConfiguredHookMatcherGroup `json:"UserPromptSubmit"`
	ManagedDir        *string                      `json:"managedDir,omitempty"`
	WindowsManagedDir *string                      `json:"windowsManagedDir,omitempty"`
}

type MarketplaceAddResponse struct {
	AlreadyAdded    bool            `json:"alreadyAdded"`
	InstalledRoot   AbsolutePathBuf `json:"installedRoot"`
	MarketplaceName string          `json:"marketplaceName"`
}

type MarketplaceInterface struct {
	DisplayName *string `json:"displayName,omitempty"`
}

type MarketplaceLoadErrorInfo struct {
	MarketplacePath AbsolutePathBuf `json:"marketplacePath"`
	Message         string          `json:"message"`
}

type MarketplaceRemoveResponse struct {
	InstalledRoot   *AbsolutePathBuf `json:"installedRoot,omitempty"`
	MarketplaceName string           `json:"marketplaceName"`
}

type MarketplaceUpgradeErrorInfo struct {
	MarketplaceName string `json:"marketplaceName"`
	Message         string `json:"message"`
}

type MarketplaceUpgradeResponse struct {
	Errors               []MarketplaceUpgradeErrorInfo `json:"errors"`
	SelectedMarketplaces []string                      `json:"selectedMarketplaces"`
	UpgradedRoots        []AbsolutePathBuf             `json:"upgradedRoots"`
}

type McpElicitationArrayType string

const (
	McpElicitationArrayTypeArray McpElicitationArrayType = "array"
)

type McpElicitationBooleanSchema struct {
	Default     *bool                     `json:"default,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Title       *string                   `json:"title,omitempty"`
	Type        McpElicitationBooleanType `json:"type"`
}

type McpElicitationBooleanType string

const (
	McpElicitationBooleanTypeBoolean McpElicitationBooleanType = "boolean"
)

type McpElicitationConstOption struct {
	Const string `json:"const"`
	Title string `json:"title"`
}

type McpElicitationEnumSchema struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *McpElicitationEnumSchema) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u McpElicitationEnumSchema) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type McpElicitationLegacyTitledEnumSchema struct {
	Default     *string                  `json:"default,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Enum        []string                 `json:"enum"`
	EnumNames   []string                 `json:"enumNames,omitempty"`
	Title       *string                  `json:"title,omitempty"`
	Type        McpElicitationStringType `json:"type"`
}

type McpElicitationStringType string

const (
	McpElicitationStringTypeString McpElicitationStringType = "string"
)

type McpElicitationMultiSelectEnumSchema struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *McpElicitationMultiSelectEnumSchema) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u McpElicitationMultiSelectEnumSchema) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type McpElicitationNumberSchema struct {
	Default     any                      `json:"default,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Maximum     any                      `json:"maximum,omitempty"`
	Minimum     any                      `json:"minimum,omitempty"`
	Title       *string                  `json:"title,omitempty"`
	Type        McpElicitationNumberType `json:"type"`
}

type McpElicitationNumberType string

const (
	McpElicitationNumberTypeNumber  McpElicitationNumberType = "number"
	McpElicitationNumberTypeInteger McpElicitationNumberType = "integer"
)

type McpElicitationObjectType string

const (
	McpElicitationObjectTypeObject McpElicitationObjectType = "object"
)

type McpElicitationPrimitiveSchema struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *McpElicitationPrimitiveSchema) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u McpElicitationPrimitiveSchema) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type McpElicitationSchema struct {
	Schema     *string                                  `json:"$schema,omitempty"`
	Properties map[string]McpElicitationPrimitiveSchema `json:"properties"`
	Required   []string                                 `json:"required,omitempty"`
	Type       McpElicitationObjectType                 `json:"type"`
}

type McpElicitationSingleSelectEnumSchema struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *McpElicitationSingleSelectEnumSchema) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u McpElicitationSingleSelectEnumSchema) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type McpElicitationStringFormat string

const (
	McpElicitationStringFormatEmail    McpElicitationStringFormat = "email"
	McpElicitationStringFormatUri      McpElicitationStringFormat = "uri"
	McpElicitationStringFormatDate     McpElicitationStringFormat = "date"
	McpElicitationStringFormatDateTime McpElicitationStringFormat = "date-time"
)

type McpElicitationStringSchema struct {
	Default     *string                     `json:"default,omitempty"`
	Description *string                     `json:"description,omitempty"`
	Format      *McpElicitationStringFormat `json:"format,omitempty"`
	MaxLength   *int64                      `json:"maxLength,omitempty"`
	MinLength   *int64                      `json:"minLength,omitempty"`
	Title       *string                     `json:"title,omitempty"`
	Type        McpElicitationStringType    `json:"type"`
}

type McpElicitationTitledEnumItems struct {
	AnyOf []McpElicitationConstOption `json:"anyOf"`
}

type McpElicitationTitledMultiSelectEnumSchema struct {
	Default     []string                      `json:"default,omitempty"`
	Description *string                       `json:"description,omitempty"`
	Items       McpElicitationTitledEnumItems `json:"items"`
	MaxItems    *int64                        `json:"maxItems,omitempty"`
	MinItems    *int64                        `json:"minItems,omitempty"`
	Title       *string                       `json:"title,omitempty"`
	Type        McpElicitationArrayType       `json:"type"`
}

type McpElicitationTitledSingleSelectEnumSchema struct {
	Default     *string                     `json:"default,omitempty"`
	Description *string                     `json:"description,omitempty"`
	OneOf       []McpElicitationConstOption `json:"oneOf"`
	Title       *string                     `json:"title,omitempty"`
	Type        McpElicitationStringType    `json:"type"`
}

type McpElicitationUntitledEnumItems struct {
	Enum []string                 `json:"enum"`
	Type McpElicitationStringType `json:"type"`
}

type McpElicitationUntitledMultiSelectEnumSchema struct {
	Default     []string                        `json:"default,omitempty"`
	Description *string                         `json:"description,omitempty"`
	Items       McpElicitationUntitledEnumItems `json:"items"`
	MaxItems    *int64                          `json:"maxItems,omitempty"`
	MinItems    *int64                          `json:"minItems,omitempty"`
	Title       *string                         `json:"title,omitempty"`
	Type        McpElicitationArrayType         `json:"type"`
}

type McpElicitationUntitledSingleSelectEnumSchema struct {
	Default     *string                  `json:"default,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Enum        []string                 `json:"enum"`
	Title       *string                  `json:"title,omitempty"`
	Type        McpElicitationStringType `json:"type"`
}

type McpResourceReadResponse struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceContent struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ResourceContent) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "uri") {
		var v ResourceContentVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "blob", "uri") {
		var v ResourceContentVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ResourceContent) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ResourceContentVariant1 struct {
	Meta     any     `json:"_meta,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	Text     string  `json:"text"`
	Uri      string  `json:"uri"`
}

type ResourceContentVariant2 struct {
	Meta     any     `json:"_meta,omitempty"`
	Blob     string  `json:"blob"`
	MimeType *string `json:"mimeType,omitempty"`
	Uri      string  `json:"uri"`
}

type McpServerElicitationAction string

const (
	McpServerElicitationActionAccept  McpServerElicitationAction = "accept"
	McpServerElicitationActionDecline McpServerElicitationAction = "decline"
	McpServerElicitationActionCancel  McpServerElicitationAction = "cancel"
)

type McpServerElicitationRequestParams struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *McpServerElicitationRequestParams) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonLooksObject(data) && jsonHasKeys(data, "message", "mode", "requestedSchema") {
		var v McpServerElicitationRequestParamsVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "message", "mode", "requestedSchema") {
		var v McpServerElicitationRequestParamsVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "elicitationId", "message", "mode", "url") {
		var v McpServerElicitationRequestParamsVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u McpServerElicitationRequestParams) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type McpServerElicitationRequestParamsVariant1 struct {
	Meta            any                  `json:"_meta,omitempty"`
	Message         string               `json:"message"`
	Mode            string               `json:"mode"`
	RequestedSchema McpElicitationSchema `json:"requestedSchema"`
}

type McpServerElicitationRequestParamsVariant2 struct {
	Meta            any    `json:"_meta,omitempty"`
	Message         string `json:"message"`
	Mode            string `json:"mode"`
	RequestedSchema any    `json:"requestedSchema"`
}

type McpServerElicitationRequestParamsVariant3 struct {
	Meta          any    `json:"_meta,omitempty"`
	ElicitationID string `json:"elicitationId"`
	Message       string `json:"message"`
	Mode          string `json:"mode"`
	Url           string `json:"url"`
}

type McpServerElicitationRequestResponse struct {
	Meta    any                        `json:"_meta,omitempty"`
	Action  McpServerElicitationAction `json:"action"`
	Content any                        `json:"content,omitempty"`
}

type McpServerOauthLoginCompletedNotification struct {
	Error    *string `json:"error,omitempty"`
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	ThreadID *string `json:"threadId,omitempty"`
}

type McpServerOauthLoginResponse struct {
	AuthorizationUrl string `json:"authorizationUrl"`
}

type McpServerRefreshResponse struct {
}

type McpServerStartupFailureReason string

const (
	McpServerStartupFailureReasonReauthenticationRequired McpServerStartupFailureReason = "reauthenticationRequired"
)

type McpServerStartupState string

const (
	McpServerStartupStateStarting  McpServerStartupState = "starting"
	McpServerStartupStateReady     McpServerStartupState = "ready"
	McpServerStartupStateFailed    McpServerStartupState = "failed"
	McpServerStartupStateCancelled McpServerStartupState = "cancelled"
)

type McpServerStatusUpdatedNotification struct {
	Error         *string                        `json:"error,omitempty"`
	FailureReason *McpServerStartupFailureReason `json:"failureReason,omitempty"`
	Name          string                         `json:"name"`
	Status        McpServerStartupState          `json:"status"`
	ThreadID      *string                        `json:"threadId,omitempty"`
}

type McpServerToolCallResponse struct {
	Meta              any   `json:"_meta,omitempty"`
	Content           []any `json:"content"`
	IsError           *bool `json:"isError,omitempty"`
	StructuredContent any   `json:"structuredContent,omitempty"`
}

type McpToolCallProgressNotification struct {
	ItemID   string `json:"itemId"`
	Message  string `json:"message"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type Model struct {
	AdditionalSpeedTiers      []string                `json:"additionalSpeedTiers,omitempty"`
	AvailabilityNux           *ModelAvailabilityNux   `json:"availabilityNux,omitempty"`
	DefaultReasoningEffort    ReasoningEffort         `json:"defaultReasoningEffort"`
	DefaultServiceTier        *string                 `json:"defaultServiceTier,omitempty"`
	Description               string                  `json:"description"`
	DisplayName               string                  `json:"displayName"`
	Hidden                    bool                    `json:"hidden"`
	ID                        string                  `json:"id"`
	InputModalities           []InputModality         `json:"inputModalities,omitempty"`
	IsDefault                 bool                    `json:"isDefault"`
	Model                     string                  `json:"model"`
	ServiceTiers              []ModelServiceTier      `json:"serviceTiers,omitempty"`
	SupportedReasoningEfforts []ReasoningEffortOption `json:"supportedReasoningEfforts"`
	SupportsPersonality       *bool                   `json:"supportsPersonality,omitempty"`
	Upgrade                   *string                 `json:"upgrade,omitempty"`
	UpgradeInfo               *ModelUpgradeInfo       `json:"upgradeInfo,omitempty"`
}

type ModelAvailabilityNux struct {
	Message string `json:"message"`
}

type ModelServiceTier struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Name        string `json:"name"`
}

type ReasoningEffortOption struct {
	Description     string          `json:"description"`
	ReasoningEffort ReasoningEffort `json:"reasoningEffort"`
}

type ModelUpgradeInfo struct {
	MigrationMarkdown *string `json:"migrationMarkdown,omitempty"`
	Model             string  `json:"model"`
	ModelLink         *string `json:"modelLink,omitempty"`
	UpgradeCopy       *string `json:"upgradeCopy,omitempty"`
}

type ModelListResponse struct {
	Data       []Model `json:"data"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

type ModelProviderCapabilitiesReadResponse struct {
	ImageGeneration bool `json:"imageGeneration"`
	NamespaceTools  bool `json:"namespaceTools"`
	WebSearch       bool `json:"webSearch"`
}

type ModelRerouteReason string

const (
	ModelRerouteReasonHighRiskCyberActivity ModelRerouteReason = "highRiskCyberActivity"
)

type ModelReroutedNotification struct {
	FromModel string             `json:"fromModel"`
	Reason    ModelRerouteReason `json:"reason"`
	ThreadID  string             `json:"threadId"`
	ToModel   string             `json:"toModel"`
	TurnID    string             `json:"turnId"`
}

type ModelSafetyBufferingUpdatedNotification struct {
	FasterModel     *string  `json:"fasterModel,omitempty"`
	Model           string   `json:"model"`
	Reasons         []string `json:"reasons"`
	ShowBufferingUi bool     `json:"showBufferingUi"`
	ThreadID        string   `json:"threadId"`
	TurnID          string   `json:"turnId"`
	UseCases        []string `json:"useCases"`
}

type ModelVerification string

const (
	ModelVerificationTrustedAccessForCyber ModelVerification = "trustedAccessForCyber"
)

type ModelVerificationNotification struct {
	ThreadID      string              `json:"threadId"`
	TurnID        string              `json:"turnId"`
	Verifications []ModelVerification `json:"verifications"`
}

type MultiAgentMode struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *MultiAgentMode) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "explicitRequestOnly", "proactive") {
		var v MultiAgentModeVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "custom") {
		var v CustomMultiAgentMode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u MultiAgentMode) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type MultiAgentModeVariant1 string

const (
	MultiAgentModeVariant1ExplicitRequestOnly MultiAgentModeVariant1 = "explicitRequestOnly"
	MultiAgentModeVariant1Proactive           MultiAgentModeVariant1 = "proactive"
)

type CustomMultiAgentMode struct {
	Custom string `json:"custom"`
}

type NetworkDomainPermission string

const (
	NetworkDomainPermissionAllow NetworkDomainPermission = "allow"
	NetworkDomainPermissionDeny  NetworkDomainPermission = "deny"
)

type NetworkRequirements struct {
	AllowLocalBinding                *bool                                  `json:"allowLocalBinding,omitempty"`
	AllowUnixSockets                 []string                               `json:"allowUnixSockets,omitempty"`
	AllowUpstreamProxy               *bool                                  `json:"allowUpstreamProxy,omitempty"`
	AllowedDomains                   []string                               `json:"allowedDomains,omitempty"`
	DangerouslyAllowAllUnixSockets   *bool                                  `json:"dangerouslyAllowAllUnixSockets,omitempty"`
	DangerouslyAllowNonLoopbackProxy *bool                                  `json:"dangerouslyAllowNonLoopbackProxy,omitempty"`
	DeniedDomains                    []string                               `json:"deniedDomains,omitempty"`
	Domains                          map[string]NetworkDomainPermission     `json:"domains,omitempty"`
	Enabled                          *bool                                  `json:"enabled,omitempty"`
	HttpPort                         *int64                                 `json:"httpPort,omitempty"`
	ManagedAllowedDomainsOnly        *bool                                  `json:"managedAllowedDomainsOnly,omitempty"`
	SocksPort                        *int64                                 `json:"socksPort,omitempty"`
	UnixSockets                      map[string]NetworkUnixSocketPermission `json:"unixSockets,omitempty"`
}

type NetworkUnixSocketPermission string

const (
	NetworkUnixSocketPermissionAllow NetworkUnixSocketPermission = "allow"
	NetworkUnixSocketPermissionDeny  NetworkUnixSocketPermission = "deny"
)

type NonSteerableTurnKind string

const (
	NonSteerableTurnKindReview  NonSteerableTurnKind = "review"
	NonSteerableTurnKindCompact NonSteerableTurnKind = "compact"
)

type PathUri string

type PermissionGrantScope string

const (
	PermissionGrantScopeTurn    PermissionGrantScope = "turn"
	PermissionGrantScopeSession PermissionGrantScope = "session"
)

type PermissionProfileListResponse struct {
	Data       []PermissionProfileSummary `json:"data"`
	NextCursor *string                    `json:"nextCursor,omitempty"`
}

type PermissionProfileSummary struct {
	Allowed     bool    `json:"allowed"`
	Description *string `json:"description,omitempty"`
	ID          string  `json:"id"`
}

type PermissionsRequestApprovalParams struct {
	Cwd           AbsolutePathBuf          `json:"cwd"`
	EnvironmentID *string                  `json:"environmentId,omitempty"`
	ItemID        string                   `json:"itemId"`
	Permissions   RequestPermissionProfile `json:"permissions"`
	Reason        *string                  `json:"reason,omitempty"`
	StartedAtMs   int64                    `json:"startedAtMs"`
	ThreadID      string                   `json:"threadId"`
	TurnID        string                   `json:"turnId"`
}

type PermissionsRequestApprovalResponse struct {
	Permissions      GrantedPermissionProfile `json:"permissions"`
	Scope            *PermissionGrantScope    `json:"scope,omitempty"`
	StrictAutoReview *bool                    `json:"strictAutoReview,omitempty"`
}

type PlanDeltaNotification struct {
	Delta    string `json:"delta"`
	ItemID   string `json:"itemId"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type PluginAuthPolicy string

const (
	PluginAuthPolicyONINSTALL PluginAuthPolicy = "ON_INSTALL"
	PluginAuthPolicyONUSE     PluginAuthPolicy = "ON_USE"
)

type PluginAvailability struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *PluginAvailability) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "DISABLED_BY_ADMIN") {
		var v PluginAvailabilityVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "AVAILABLE") {
		var v PluginAvailabilityVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u PluginAvailability) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type PluginAvailabilityVariant1 string

const (
	PluginAvailabilityVariant1DISABLEDBYADMIN PluginAvailabilityVariant1 = "DISABLED_BY_ADMIN"
)

type PluginAvailabilityVariant2 string

const (
	PluginAvailabilityVariant2AVAILABLE PluginAvailabilityVariant2 = "AVAILABLE"
)

type PluginDetail struct {
	AppTemplates    []AppTemplateSummary   `json:"appTemplates"`
	Apps            []AppSummary           `json:"apps"`
	Description     *string                `json:"description,omitempty"`
	Hooks           []PluginHookSummary    `json:"hooks"`
	MarketplaceName string                 `json:"marketplaceName"`
	MarketplacePath *AbsolutePathBuf       `json:"marketplacePath,omitempty"`
	McpServers      []string               `json:"mcpServers"`
	ScheduledTasks  []ScheduledTaskSummary `json:"scheduledTasks,omitempty"`
	ShareUrl        *string                `json:"shareUrl,omitempty"`
	Skills          []SkillSummary         `json:"skills"`
	Summary         PluginSummary          `json:"summary"`
}

type PluginHookSummary struct {
	EventName HookEventName `json:"eventName"`
	Key       string        `json:"key"`
}

type ScheduledTaskSummary struct {
	Key      string                `json:"key"`
	Name     string                `json:"name"`
	Prompt   string                `json:"prompt"`
	Schedule ScheduledTaskSchedule `json:"schedule"`
}

type ScheduledTaskSchedule struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ScheduledTaskSchedule) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "hourly":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "intervalHours", "type")) {
					break
				}
				var v HourlyScheduledTaskSchedule
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "daily":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "time", "type")) {
					break
				}
				var v DailyScheduledTaskSchedule
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "weekdays":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "time", "type")) {
					break
				}
				var v WeekdaysScheduledTaskSchedule
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "weekly":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "days", "time", "type")) {
					break
				}
				var v WeeklyScheduledTaskSchedule
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "intervalHours", "type") {
		var v HourlyScheduledTaskSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "time", "type") {
		var v DailyScheduledTaskSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "time", "type") {
		var v WeekdaysScheduledTaskSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "days", "time", "type") {
		var v WeeklyScheduledTaskSchedule
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ScheduledTaskSchedule) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type HourlyScheduledTaskSchedule struct {
	Days          []ScheduledTaskWeekday `json:"days,omitempty"`
	IntervalHours int64                  `json:"intervalHours"`
	Type          string                 `json:"type"`
}

type ScheduledTaskWeekday string

const (
	ScheduledTaskWeekdayMO ScheduledTaskWeekday = "MO"
	ScheduledTaskWeekdayTU ScheduledTaskWeekday = "TU"
	ScheduledTaskWeekdayWE ScheduledTaskWeekday = "WE"
	ScheduledTaskWeekdayTH ScheduledTaskWeekday = "TH"
	ScheduledTaskWeekdayFR ScheduledTaskWeekday = "FR"
	ScheduledTaskWeekdaySA ScheduledTaskWeekday = "SA"
	ScheduledTaskWeekdaySU ScheduledTaskWeekday = "SU"
)

type DailyScheduledTaskSchedule struct {
	Time string `json:"time"`
	Type string `json:"type"`
}

type WeekdaysScheduledTaskSchedule struct {
	Time string `json:"time"`
	Type string `json:"type"`
}

type WeeklyScheduledTaskSchedule struct {
	Days []ScheduledTaskWeekday `json:"days"`
	Time string                 `json:"time"`
	Type string                 `json:"type"`
}

type SkillSummary struct {
	Description      string           `json:"description"`
	Enabled          bool             `json:"enabled"`
	Interface        *SkillInterface  `json:"interface,omitempty"`
	Name             string           `json:"name"`
	Path             *AbsolutePathBuf `json:"path,omitempty"`
	ShortDescription *string          `json:"shortDescription,omitempty"`
}

type SkillInterface struct {
	BrandColor       *string          `json:"brandColor,omitempty"`
	DefaultPrompt    *string          `json:"defaultPrompt,omitempty"`
	DisplayName      *string          `json:"displayName,omitempty"`
	IconLarge        *AbsolutePathBuf `json:"iconLarge,omitempty"`
	IconLargeUrl     *string          `json:"iconLargeUrl,omitempty"`
	IconSmall        *AbsolutePathBuf `json:"iconSmall,omitempty"`
	IconSmallUrl     *string          `json:"iconSmallUrl,omitempty"`
	ShortDescription *string          `json:"shortDescription,omitempty"`
}

type PluginSummary struct {
	AuthPolicy                       PluginAuthPolicy           `json:"authPolicy"`
	Availability                     *PluginAvailability        `json:"availability,omitempty"`
	DisabledReason                   *PluginDisabledReason      `json:"disabledReason,omitempty"`
	EligiblePlanTypes                []string                   `json:"eligiblePlanTypes,omitempty"`
	Enabled                          bool                       `json:"enabled"`
	ID                               string                     `json:"id"`
	InstallPolicy                    PluginInstallPolicy        `json:"installPolicy"`
	InstallPolicySource              *PluginInstallPolicySource `json:"installPolicySource,omitempty"`
	Installed                        bool                       `json:"installed"`
	InstalledAt                      *int64                     `json:"installedAt,omitempty"`
	Interface                        *PluginInterface           `json:"interface,omitempty"`
	Keywords                         []string                   `json:"keywords,omitempty"`
	LocalVersion                     *string                    `json:"localVersion,omitempty"`
	MustShowInstallationInterstitial *bool                      `json:"mustShowInstallationInterstitial,omitempty"`
	Name                             string                     `json:"name"`
	RemotePluginID                   *string                    `json:"remotePluginId,omitempty"`
	ShareContext                     *PluginShareContext        `json:"shareContext,omitempty"`
	Source                           PluginSource               `json:"source"`
	Version                          *string                    `json:"version,omitempty"`
}

type PluginDisabledReason string

const (
	PluginDisabledReasonDisabledByAdmin        PluginDisabledReason = "disabled_by_admin"
	PluginDisabledReasonPlanNotEligible        PluginDisabledReason = "plan_not_eligible"
	PluginDisabledReasonRequiredAppUnavailable PluginDisabledReason = "required_app_unavailable"
	PluginDisabledReasonUnknown                PluginDisabledReason = "unknown"
)

type PluginInstallPolicy string

const (
	PluginInstallPolicyNOTAVAILABLE       PluginInstallPolicy = "NOT_AVAILABLE"
	PluginInstallPolicyAVAILABLE          PluginInstallPolicy = "AVAILABLE"
	PluginInstallPolicyINSTALLEDBYDEFAULT PluginInstallPolicy = "INSTALLED_BY_DEFAULT"
)

type PluginInstallPolicySource string

const (
	PluginInstallPolicySourceWORKSPACESETTING     PluginInstallPolicySource = "WORKSPACE_SETTING"
	PluginInstallPolicySourceIMPLICITCANONICALAPP PluginInstallPolicySource = "IMPLICIT_CANONICAL_APP"
)

type PluginInterface struct {
	BrandColor        *string           `json:"brandColor,omitempty"`
	Capabilities      []string          `json:"capabilities"`
	Category          *string           `json:"category,omitempty"`
	ComposerIcon      *AbsolutePathBuf  `json:"composerIcon,omitempty"`
	ComposerIconUrl   *string           `json:"composerIconUrl,omitempty"`
	DefaultPrompt     []string          `json:"defaultPrompt,omitempty"`
	DeveloperName     *string           `json:"developerName,omitempty"`
	DisplayName       *string           `json:"displayName,omitempty"`
	Logo              *AbsolutePathBuf  `json:"logo,omitempty"`
	LogoDark          *AbsolutePathBuf  `json:"logoDark,omitempty"`
	LogoUrl           *string           `json:"logoUrl,omitempty"`
	LogoUrlDark       *string           `json:"logoUrlDark,omitempty"`
	LongDescription   *string           `json:"longDescription,omitempty"`
	PrivacyPolicyUrl  *string           `json:"privacyPolicyUrl,omitempty"`
	ScreenshotUrls    []string          `json:"screenshotUrls"`
	Screenshots       []AbsolutePathBuf `json:"screenshots"`
	ShortDescription  *string           `json:"shortDescription,omitempty"`
	TermsOfServiceUrl *string           `json:"termsOfServiceUrl,omitempty"`
	WebsiteUrl        *string           `json:"websiteUrl,omitempty"`
}

type PluginShareContext struct {
	CanPublishToWorkspace *bool                       `json:"canPublishToWorkspace,omitempty"`
	CreatorAccountUserID  *string                     `json:"creatorAccountUserId,omitempty"`
	CreatorName           *string                     `json:"creatorName,omitempty"`
	Discoverability       *PluginShareDiscoverability `json:"discoverability,omitempty"`
	RemotePluginID        string                      `json:"remotePluginId"`
	RemoteVersion         *string                     `json:"remoteVersion,omitempty"`
	SharePrincipals       []PluginSharePrincipal      `json:"sharePrincipals,omitempty"`
	ShareUrl              *string                     `json:"shareUrl,omitempty"`
}

type PluginSharePrincipal struct {
	Name          string                   `json:"name"`
	PrincipalID   string                   `json:"principalId"`
	PrincipalType PluginSharePrincipalType `json:"principalType"`
	Role          PluginSharePrincipalRole `json:"role"`
}

type PluginSharePrincipalRole string

const (
	PluginSharePrincipalRoleReader PluginSharePrincipalRole = "reader"
	PluginSharePrincipalRoleEditor PluginSharePrincipalRole = "editor"
	PluginSharePrincipalRoleOwner  PluginSharePrincipalRole = "owner"
)

type PluginSource struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *PluginSource) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "local":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "path", "type")) {
					break
				}
				var v LocalPluginSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "git":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type", "url")) {
					break
				}
				var v GitPluginSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "npm":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "package", "type")) {
					break
				}
				var v NpmPluginSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "remote":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v RemotePluginSource
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "path", "type") {
		var v LocalPluginSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type", "url") {
		var v GitPluginSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "package", "type") {
		var v NpmPluginSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v RemotePluginSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u PluginSource) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type LocalPluginSource struct {
	Path AbsolutePathBuf `json:"path"`
	Type string          `json:"type"`
}

type GitPluginSource struct {
	Path    *string `json:"path,omitempty"`
	RefName *string `json:"refName,omitempty"`
	Sha     *string `json:"sha,omitempty"`
	Type    string  `json:"type"`
	Url     string  `json:"url"`
}

type NpmPluginSource struct {
	Package  string  `json:"package"`
	Registry *string `json:"registry,omitempty"`
	Type     string  `json:"type"`
	Version  *string `json:"version,omitempty"`
}

type RemotePluginSource struct {
	Type string `json:"type"`
}

type PluginInstallResponse struct {
	AppsNeedingAuth []AppSummary     `json:"appsNeedingAuth"`
	AuthPolicy      PluginAuthPolicy `json:"authPolicy"`
}

type PluginInstalledResponse struct {
	MarketplaceLoadErrors []MarketplaceLoadErrorInfo `json:"marketplaceLoadErrors,omitempty"`
	Marketplaces          []PluginMarketplaceEntry   `json:"marketplaces"`
}

type PluginMarketplaceEntry struct {
	Interface *MarketplaceInterface `json:"interface,omitempty"`
	Name      string                `json:"name"`
	Path      *AbsolutePathBuf      `json:"path,omitempty"`
	Plugins   []PluginSummary       `json:"plugins"`
}

type PluginListResponse struct {
	FeaturedPluginIds     []string                   `json:"featuredPluginIds,omitempty"`
	MarketplaceLoadErrors []MarketplaceLoadErrorInfo `json:"marketplaceLoadErrors,omitempty"`
	Marketplaces          []PluginMarketplaceEntry   `json:"marketplaces"`
}

type PluginReadResponse struct {
	Plugin PluginDetail `json:"plugin"`
}

type PluginSearchResult struct {
	MarketplaceName string           `json:"marketplaceName"`
	MarketplacePath *AbsolutePathBuf `json:"marketplacePath,omitempty"`
	Plugin          PluginSummary    `json:"plugin"`
}

type PluginSearchScope string

const (
	PluginSearchScopeGlobal    PluginSearchScope = "global"
	PluginSearchScopeWorkspace PluginSearchScope = "workspace"
	PluginSearchScopePersonal  PluginSearchScope = "personal"
)

type PluginShareCheckoutResponse struct {
	MarketplaceName string          `json:"marketplaceName"`
	MarketplacePath AbsolutePathBuf `json:"marketplacePath"`
	PluginID        string          `json:"pluginId"`
	PluginName      string          `json:"pluginName"`
	PluginPath      AbsolutePathBuf `json:"pluginPath"`
	RemotePluginID  string          `json:"remotePluginId"`
	RemoteVersion   *string         `json:"remoteVersion,omitempty"`
}

type PluginShareDeleteResponse struct {
}

type PluginShareListItem struct {
	LocalPluginPath *AbsolutePathBuf `json:"localPluginPath,omitempty"`
	Plugin          PluginSummary    `json:"plugin"`
}

type PluginShareListResponse struct {
	Data []PluginShareListItem `json:"data"`
}

type PluginShareSaveResponse struct {
	CanPublishToWorkspace *bool  `json:"canPublishToWorkspace,omitempty"`
	RemotePluginID        string `json:"remotePluginId"`
	ShareUrl              string `json:"shareUrl"`
}

type PluginShareUpdateTargetsResponse struct {
	Discoverability PluginShareDiscoverability `json:"discoverability"`
	Principals      []PluginSharePrincipal     `json:"principals"`
}

type PluginSkillReadResponse struct {
	Contents *string `json:"contents,omitempty"`
}

type PluginUninstallResponse struct {
}

type ProcessExitedNotification struct {
	ExitCode         int64  `json:"exitCode"`
	ProcessHandle    string `json:"processHandle"`
	Stderr           string `json:"stderr"`
	StderrCapReached bool   `json:"stderrCapReached"`
	Stdout           string `json:"stdout"`
	StdoutCapReached bool   `json:"stdoutCapReached"`
}

type ProcessOutputDeltaNotification struct {
	CapReached    bool                `json:"capReached"`
	DeltaBase64   string              `json:"deltaBase64"`
	ProcessHandle string              `json:"processHandle"`
	Stream        ProcessOutputStream `json:"stream"`
}

type ProcessOutputStream struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ProcessOutputStream) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "stdout") {
		var v ProcessOutputStreamVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "stderr") {
		var v ProcessOutputStreamVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ProcessOutputStream) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ProcessOutputStreamVariant1 string

const (
	ProcessOutputStreamVariant1Stdout ProcessOutputStreamVariant1 = "stdout"
)

type ProcessOutputStreamVariant2 string

const (
	ProcessOutputStreamVariant2Stderr ProcessOutputStreamVariant2 = "stderr"
)

type ProcessTerminalSize struct {
	Cols int64 `json:"cols"`
	Rows int64 `json:"rows"`
}

type RawResponseCompletedNotification struct {
	ResponseID string               `json:"responseId"`
	ThreadID   string               `json:"threadId"`
	TurnID     string               `json:"turnId"`
	Usage      *TokenUsageBreakdown `json:"usage,omitempty"`
}

type TokenUsageBreakdown struct {
	CacheWriteInputTokens *int64 `json:"cacheWriteInputTokens,omitempty"`
	CachedInputTokens     int64  `json:"cachedInputTokens"`
	InputTokens           int64  `json:"inputTokens"`
	OutputTokens          int64  `json:"outputTokens"`
	ReasoningOutputTokens int64  `json:"reasoningOutputTokens"`
	TotalTokens           int64  `json:"totalTokens"`
}

type RawResponseItemCompletedNotification struct {
	Item     ResponseItem `json:"item"`
	ThreadID string       `json:"threadId"`
	TurnID   string       `json:"turnId"`
}

type ResponseItem struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ResponseItem) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "message":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "content", "role", "type")) {
					break
				}
				var v MessageResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "agent_message":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "author", "content", "recipient", "type")) {
					break
				}
				var v AgentMessageResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "reasoning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "summary", "type")) {
					break
				}
				var v ReasoningResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "local_shell_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "action", "status", "type")) {
					break
				}
				var v LocalShellCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "function_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "arguments", "call_id", "name", "type")) {
					break
				}
				var v FunctionCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "tool_search_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "arguments", "execution", "type")) {
					break
				}
				var v ToolSearchCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "function_call_output":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "call_id", "output", "type")) {
					break
				}
				var v FunctionCallOutputResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "custom_tool_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "call_id", "input", "name", "type")) {
					break
				}
				var v CustomToolCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "custom_tool_call_output":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "call_id", "output", "type")) {
					break
				}
				var v CustomToolCallOutputResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "tool_search_output":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "execution", "status", "tools", "type")) {
					break
				}
				var v ToolSearchOutputResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "web_search_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v WebSearchCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "image_generation_call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "result", "status", "type")) {
					break
				}
				var v ImageGenerationCallResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "compaction":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type")) {
					break
				}
				var v CompactionResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "compaction_trigger":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v CompactionTriggerResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "context_compaction":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v ContextCompactionResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "other":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v OtherResponseItem
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "content", "role", "type") {
		var v MessageResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "author", "content", "recipient", "type") {
		var v AgentMessageResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "summary", "type") {
		var v ReasoningResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "action", "status", "type") {
		var v LocalShellCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "arguments", "call_id", "name", "type") {
		var v FunctionCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "arguments", "execution", "type") {
		var v ToolSearchCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "call_id", "output", "type") {
		var v FunctionCallOutputResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "call_id", "input", "name", "type") {
		var v CustomToolCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "call_id", "output", "type") {
		var v CustomToolCallOutputResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "execution", "status", "tools", "type") {
		var v ToolSearchOutputResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v WebSearchCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "result", "status", "type") {
		var v ImageGenerationCallResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "encrypted_content", "type") {
		var v CompactionResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v CompactionTriggerResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v ContextCompactionResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v OtherResponseItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ResponseItem) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type MessageResponseItem struct {
	Content                                []ContentItem                           `json:"content"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Phase                                  *MessagePhase                           `json:"phase,omitempty"`
	Role                                   string                                  `json:"role"`
	Type                                   string                                  `json:"type"`
}

type AgentMessageResponseItem struct {
	Author                                 string                                  `json:"author"`
	Content                                []AgentMessageInputContent              `json:"content"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Recipient                              string                                  `json:"recipient"`
	Type                                   string                                  `json:"type"`
}

type ReasoningResponseItem struct {
	Content                                []ReasoningItemContent                  `json:"content,omitempty"`
	EncryptedContent                       *string                                 `json:"encrypted_content,omitempty"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Summary                                []ReasoningItemReasoningSummary         `json:"summary"`
	Type                                   string                                  `json:"type"`
}

type ReasoningItemContent struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ReasoningItemContent) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "reasoning_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v ReasoningTextReasoningItemContent
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v TextReasoningItemContent
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v ReasoningTextReasoningItemContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v TextReasoningItemContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ReasoningItemContent) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ReasoningTextReasoningItemContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type TextReasoningItemContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type ReasoningItemReasoningSummary struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ReasoningItemReasoningSummary) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "summary_text":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "text", "type")) {
					break
				}
				var v SummaryTextReasoningItemReasoningSummary
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "text", "type") {
		var v SummaryTextReasoningItemReasoningSummary
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ReasoningItemReasoningSummary) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SummaryTextReasoningItemReasoningSummary struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type LocalShellCallResponseItem struct {
	Action                                 LocalShellAction                        `json:"action"`
	CallID                                 *string                                 `json:"call_id,omitempty"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Status                                 LocalShellStatus                        `json:"status"`
	Type                                   string                                  `json:"type"`
}

type FunctionCallResponseItem struct {
	Arguments                              string                                  `json:"arguments"`
	CallID                                 string                                  `json:"call_id"`
	EncryptedFunctionArgs                  []string                                `json:"encrypted_function_args,omitempty"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Name                                   string                                  `json:"name"`
	Namespace                              *string                                 `json:"namespace,omitempty"`
	Type                                   string                                  `json:"type"`
}

type ToolSearchCallResponseItem struct {
	Arguments                              any                                     `json:"arguments"`
	CallID                                 *string                                 `json:"call_id,omitempty"`
	Execution                              string                                  `json:"execution"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Status                                 *string                                 `json:"status,omitempty"`
	Type                                   string                                  `json:"type"`
}

type FunctionCallOutputResponseItem struct {
	CallID                                 string                                  `json:"call_id"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Output                                 FunctionCallOutputBody                  `json:"output"`
	Type                                   string                                  `json:"type"`
}

type CustomToolCallResponseItem struct {
	CallID                                 string                                  `json:"call_id"`
	ID                                     *string                                 `json:"id,omitempty"`
	Input                                  string                                  `json:"input"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Name                                   string                                  `json:"name"`
	Namespace                              *string                                 `json:"namespace,omitempty"`
	Status                                 *string                                 `json:"status,omitempty"`
	Type                                   string                                  `json:"type"`
}

type CustomToolCallOutputResponseItem struct {
	CallID                                 string                                  `json:"call_id"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Name                                   *string                                 `json:"name,omitempty"`
	Output                                 FunctionCallOutputBody                  `json:"output"`
	Type                                   string                                  `json:"type"`
}

type ToolSearchOutputResponseItem struct {
	CallID                                 *string                                 `json:"call_id,omitempty"`
	Execution                              string                                  `json:"execution"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Status                                 string                                  `json:"status"`
	Tools                                  []any                                   `json:"tools"`
	Type                                   string                                  `json:"type"`
}

type WebSearchCallResponseItem struct {
	Action                                 *ResponsesApiWebSearchAction            `json:"action,omitempty"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Status                                 *string                                 `json:"status,omitempty"`
	Type                                   string                                  `json:"type"`
}

type ResponsesApiWebSearchAction struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ResponsesApiWebSearchAction) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "search":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v SearchResponsesApiWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "open_page":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v OpenPageResponsesApiWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "find_in_page":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v FindInPageResponsesApiWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "other":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v OtherResponsesApiWebSearchAction
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v SearchResponsesApiWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v OpenPageResponsesApiWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v FindInPageResponsesApiWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v OtherResponsesApiWebSearchAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ResponsesApiWebSearchAction) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SearchResponsesApiWebSearchAction struct {
	Queries []string `json:"queries,omitempty"`
	Query   *string  `json:"query,omitempty"`
	Type    string   `json:"type"`
}

type OpenPageResponsesApiWebSearchAction struct {
	Type string  `json:"type"`
	Url  *string `json:"url,omitempty"`
}

type FindInPageResponsesApiWebSearchAction struct {
	Pattern *string `json:"pattern,omitempty"`
	Type    string  `json:"type"`
	Url     *string `json:"url,omitempty"`
}

type OtherResponsesApiWebSearchAction struct {
	Type string `json:"type"`
}

type ImageGenerationCallResponseItem struct {
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Result                                 string                                  `json:"result"`
	RevisedPrompt                          *string                                 `json:"revised_prompt,omitempty"`
	Status                                 string                                  `json:"status"`
	Type                                   string                                  `json:"type"`
}

type CompactionResponseItem struct {
	EncryptedContent                       string                                  `json:"encrypted_content"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Type                                   string                                  `json:"type"`
}

type CompactionTriggerResponseItem struct {
	Type string `json:"type"`
}

type ContextCompactionResponseItem struct {
	EncryptedContent                       *string                                 `json:"encrypted_content,omitempty"`
	ID                                     *string                                 `json:"id,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Type                                   string                                  `json:"type"`
}

type OtherResponseItem struct {
	Type string `json:"type"`
}

type RealtimeConversationVersion string

const (
	RealtimeConversationVersionV1 RealtimeConversationVersion = "v1"
	RealtimeConversationVersionV2 RealtimeConversationVersion = "v2"
	RealtimeConversationVersionV3 RealtimeConversationVersion = "v3"
)

type RealtimeOutputModality string

const (
	RealtimeOutputModalityText  RealtimeOutputModality = "text"
	RealtimeOutputModalityAudio RealtimeOutputModality = "audio"
)

type RealtimeVoice string

const (
	RealtimeVoiceAlloy   RealtimeVoice = "alloy"
	RealtimeVoiceArbor   RealtimeVoice = "arbor"
	RealtimeVoiceAsh     RealtimeVoice = "ash"
	RealtimeVoiceBallad  RealtimeVoice = "ballad"
	RealtimeVoiceBreeze  RealtimeVoice = "breeze"
	RealtimeVoiceCedar   RealtimeVoice = "cedar"
	RealtimeVoiceCoral   RealtimeVoice = "coral"
	RealtimeVoiceCove    RealtimeVoice = "cove"
	RealtimeVoiceEcho    RealtimeVoice = "echo"
	RealtimeVoiceEmber   RealtimeVoice = "ember"
	RealtimeVoiceJuniper RealtimeVoice = "juniper"
	RealtimeVoiceMaple   RealtimeVoice = "maple"
	RealtimeVoiceMarin   RealtimeVoice = "marin"
	RealtimeVoiceSage    RealtimeVoice = "sage"
	RealtimeVoiceShimmer RealtimeVoice = "shimmer"
	RealtimeVoiceSol     RealtimeVoice = "sol"
	RealtimeVoiceSpruce  RealtimeVoice = "spruce"
	RealtimeVoiceVale    RealtimeVoice = "vale"
	RealtimeVoiceVerse   RealtimeVoice = "verse"
)

type RealtimeVoicesList struct {
	DefaultV1 RealtimeVoice   `json:"defaultV1"`
	DefaultV2 RealtimeVoice   `json:"defaultV2"`
	V1        []RealtimeVoice `json:"v1"`
	V2        []RealtimeVoice `json:"v2"`
}

type ReasoningSummaryPartAddedNotification struct {
	ItemID       string `json:"itemId"`
	SummaryIndex int64  `json:"summaryIndex"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

type ReasoningSummaryTextDeltaNotification struct {
	Delta        string `json:"delta"`
	ItemID       string `json:"itemId"`
	SummaryIndex int64  `json:"summaryIndex"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

type ReasoningTextDeltaNotification struct {
	ContentIndex int64  `json:"contentIndex"`
	Delta        string `json:"delta"`
	ItemID       string `json:"itemId"`
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
}

type RemoteControlConnectionStatus string

const (
	RemoteControlConnectionStatusDisabled   RemoteControlConnectionStatus = "disabled"
	RemoteControlConnectionStatusConnecting RemoteControlConnectionStatus = "connecting"
	RemoteControlConnectionStatusConnected  RemoteControlConnectionStatus = "connected"
	RemoteControlConnectionStatusErrored    RemoteControlConnectionStatus = "errored"
)

type RemoteControlDisableParams struct {
	Ephemeral *bool `json:"ephemeral,omitempty"`
}

type RemoteControlEnableParams struct {
	Ephemeral *bool `json:"ephemeral,omitempty"`
}

type RemoteControlStatusChangedNotification struct {
	EnvironmentID  *string                       `json:"environmentId,omitempty"`
	InstallationID string                        `json:"installationId"`
	ServerName     string                        `json:"serverName"`
	Status         RemoteControlConnectionStatus `json:"status"`
}

type ReviewStartResponse struct {
	ReviewThreadID string `json:"reviewThreadId"`
	Turn           Turn   `json:"turn"`
}

type Turn struct {
	CompletedAt *int64         `json:"completedAt,omitempty"`
	DurationMs  *int64         `json:"durationMs,omitempty"`
	Error       *TurnError     `json:"error,omitempty"`
	ID          string         `json:"id"`
	Items       []ThreadItem   `json:"items"`
	ItemsView   *TurnItemsView `json:"itemsView,omitempty"`
	StartedAt   *int64         `json:"startedAt,omitempty"`
	Status      TurnStatus     `json:"status"`
}

type TurnItemsView struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *TurnItemsView) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "notLoaded") {
		var v TurnItemsViewVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "summary") {
		var v TurnItemsViewVariant2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonStringIn(data, "full") {
		var v TurnItemsViewVariant3
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u TurnItemsView) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type TurnItemsViewVariant1 string

const (
	TurnItemsViewVariant1NotLoaded TurnItemsViewVariant1 = "notLoaded"
)

type TurnItemsViewVariant2 string

const (
	TurnItemsViewVariant2Summary TurnItemsViewVariant2 = "summary"
)

type TurnItemsViewVariant3 string

const (
	TurnItemsViewVariant3Full TurnItemsViewVariant3 = "full"
)

type TurnStatus string

const (
	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusInterrupted TurnStatus = "interrupted"
	TurnStatusFailed      TurnStatus = "failed"
	TurnStatusInProgress  TurnStatus = "inProgress"
)

type SelectedCapabilityRoot struct {
	ID       string                 `json:"id"`
	Location CapabilityRootLocation `json:"location"`
}

type SendAddCreditsNudgeEmailResponse struct {
	Status AddCreditsNudgeEmailStatus `json:"status"`
}

type ServerNotification struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ServerNotification) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "method") {
		if disc, ok := unionDiscriminator(data, "method"); ok {
			switch disc {
			case "error":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ErrorNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/status/changed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadStatusChangedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/archived":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadArchivedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/deleted":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadDeletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/unarchived":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadUnarchivedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/closed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadClosedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "skills/changed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v SkillsChangedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/name/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadNameUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/goal/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadGoalUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/goal/cleared":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadGoalClearedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/environment/connected":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadEnvironmentConnectedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/environment/disconnected":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadEnvironmentDisconnectedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/settings/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadSettingsUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/tokenUsage/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadTokenUsageUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v TurnStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "hook/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v HookStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v TurnCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "hook/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v HookCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/diff/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v TurnDiffUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/plan/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v TurnPlanUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/autoApprovalReview/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemAutoApprovalReviewStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/autoApprovalReview/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemAutoApprovalReviewCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/agentMessage/delta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemAgentMessageDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/plan/delta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemPlanDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "command/exec/outputDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v CommandExecOutputDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "process/outputDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ProcessOutputDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "process/exited":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ProcessExitedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/commandExecution/outputDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemCommandExecutionOutputDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/commandExecution/terminalInteraction":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemCommandExecutionTerminalInteractionNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/fileChange/outputDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemFileChangeOutputDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/fileChange/patchUpdated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemFileChangePatchUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "serverRequest/resolved":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ServerRequestResolvedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/mcpToolCall/progress":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemMcpToolCallProgressNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/oauthLogin/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v McpServerOauthLoginCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/startupStatus/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v McpServerStartupStatusUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v AccountUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/rateLimits/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v AccountRateLimitsUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "app/list/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v AppListUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "remoteControl/status/changed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v RemoteControlStatusChangedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/import/progress":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ExternalAgentConfigImportProgressNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "externalAgentConfig/import/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ExternalAgentConfigImportCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fs/changed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v FsChangedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/reasoning/summaryTextDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemReasoningSummaryTextDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/reasoning/summaryPartAdded":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemReasoningSummaryPartAddedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/reasoning/textDelta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ItemReasoningTextDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/compacted":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadCompactedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "model/rerouted":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ModelReroutedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "model/verification":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ModelVerificationNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "turn/moderationMetadata":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v TurnModerationMetadataNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "model/safetyBuffering/updated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ModelSafetyBufferingUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "warning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v WarningNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "guardianWarning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v GuardianWarningNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "deprecationNotice":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v DeprecationNoticeNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "configWarning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ConfigWarningNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fuzzyFileSearch/sessionUpdated":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v FuzzyFileSearchSessionUpdatedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "fuzzyFileSearch/sessionCompleted":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v FuzzyFileSearchSessionCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/started":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeStartedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/itemAdded":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeItemAddedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/transcript/delta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeTranscriptDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/transcript/done":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeTranscriptDoneNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/outputAudio/delta":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeOutputAudioDeltaNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/sdp":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeSdpNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/error":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeErrorNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "thread/realtime/closed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v ThreadRealtimeClosedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "windows/worldWritableWarning":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v WindowsWorldWritableWarningNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "windowsSandbox/setupCompleted":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v WindowsSandboxSetupCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/login/completed":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "method", "params")) {
					break
				}
				var v AccountLoginCompletedNotificationEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ErrorNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadStatusChangedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadArchivedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadDeletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadUnarchivedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadClosedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v SkillsChangedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadNameUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadGoalUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadGoalClearedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadEnvironmentConnectedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadEnvironmentDisconnectedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadSettingsUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadTokenUsageUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v TurnStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v HookStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v TurnCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v HookCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v TurnDiffUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v TurnPlanUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemAutoApprovalReviewStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemAutoApprovalReviewCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemAgentMessageDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemPlanDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v CommandExecOutputDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ProcessOutputDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ProcessExitedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemCommandExecutionOutputDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemCommandExecutionTerminalInteractionNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemFileChangeOutputDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemFileChangePatchUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ServerRequestResolvedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemMcpToolCallProgressNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v McpServerOauthLoginCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v McpServerStartupStatusUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v AccountUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v AccountRateLimitsUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v AppListUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v RemoteControlStatusChangedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ExternalAgentConfigImportProgressNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ExternalAgentConfigImportCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v FsChangedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemReasoningSummaryTextDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemReasoningSummaryPartAddedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ItemReasoningTextDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadCompactedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ModelReroutedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ModelVerificationNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v TurnModerationMetadataNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ModelSafetyBufferingUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v WarningNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v GuardianWarningNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v DeprecationNoticeNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ConfigWarningNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v FuzzyFileSearchSessionUpdatedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v FuzzyFileSearchSessionCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeStartedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeItemAddedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeTranscriptDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeTranscriptDoneNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeOutputAudioDeltaNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeSdpNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeErrorNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v ThreadRealtimeClosedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v WindowsWorldWritableWarningNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v WindowsSandboxSetupCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "method", "params") {
		var v AccountLoginCompletedNotificationEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ServerNotification) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ErrorNotificationEnvelope struct {
	Method string            `json:"method"`
	Params ErrorNotification `json:"params"`
}

type ThreadStartedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ThreadStartedNotification `json:"params"`
}

type ThreadStartedNotification struct {
	Thread Thread `json:"thread"`
}

type Thread struct {
	AgentNickname    *string         `json:"agentNickname,omitempty"`
	AgentRole        *string         `json:"agentRole,omitempty"`
	CliVersion       string          `json:"cliVersion"`
	CreatedAt        int64           `json:"createdAt"`
	Cwd              AbsolutePathBuf `json:"cwd"`
	Ephemeral        bool            `json:"ephemeral"`
	ForkedFromID     *string         `json:"forkedFromId,omitempty"`
	GitInfo          *GitInfo        `json:"gitInfo,omitempty"`
	ID               string          `json:"id"`
	ModelProvider    string          `json:"modelProvider"`
	Name             *string         `json:"name,omitempty"`
	ParentThreadID   *string         `json:"parentThreadId,omitempty"`
	Path             *string         `json:"path,omitempty"`
	Preview          string          `json:"preview"`
	RecencyAt        *int64          `json:"recencyAt,omitempty"`
	Section          *ThreadSection  `json:"section,omitempty"`
	SectionEnteredAt *int64          `json:"sectionEnteredAt,omitempty"`
	SessionID        string          `json:"sessionId"`
	Source           SessionSource   `json:"source"`
	Status           ThreadStatus    `json:"status"`
	ThreadSource     *ThreadSource   `json:"threadSource,omitempty"`
	Turns            []Turn          `json:"turns"`
	UpdatedAt        int64           `json:"updatedAt"`
}

type ThreadSection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SessionSource struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *SessionSource) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "cli", "vscode", "exec", "appServer", "unknown") {
		var v SessionSourceVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "custom") {
		var v CustomSessionSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "subAgent") {
		var v SubAgentSessionSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u SessionSource) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SessionSourceVariant1 string

const (
	SessionSourceVariant1Cli       SessionSourceVariant1 = "cli"
	SessionSourceVariant1Vscode    SessionSourceVariant1 = "vscode"
	SessionSourceVariant1Exec      SessionSourceVariant1 = "exec"
	SessionSourceVariant1AppServer SessionSourceVariant1 = "appServer"
	SessionSourceVariant1Unknown   SessionSourceVariant1 = "unknown"
)

type CustomSessionSource struct {
	Custom string `json:"custom"`
}

type SubAgentSessionSource struct {
	SubAgent SubAgentSource `json:"subAgent"`
}

type SubAgentSource struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *SubAgentSource) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonStringIn(data, "review", "compact", "memory_consolidation") {
		var v SubAgentSourceVariant1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "thread_spawn") {
		var v ThreadSpawnSubAgentSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "other") {
		var v OtherSubAgentSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u SubAgentSource) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type SubAgentSourceVariant1 string

const (
	SubAgentSourceVariant1Review              SubAgentSourceVariant1 = "review"
	SubAgentSourceVariant1Compact             SubAgentSourceVariant1 = "compact"
	SubAgentSourceVariant1MemoryConsolidation SubAgentSourceVariant1 = "memory_consolidation"
)

type ThreadSpawnSubAgentSource struct {
	ThreadSpawn map[string]any `json:"thread_spawn"`
}

type OtherSubAgentSource struct {
	Other string `json:"other"`
}

type ThreadStatus struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ThreadStatus) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "notLoaded":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v NotLoadedThreadStatus
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "idle":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v IdleThreadStatus
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "systemError":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v SystemErrorThreadStatus
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "active":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "activeFlags", "type")) {
					break
				}
				var v ActiveThreadStatus
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v NotLoadedThreadStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v IdleThreadStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v SystemErrorThreadStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "activeFlags", "type") {
		var v ActiveThreadStatus
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ThreadStatus) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type NotLoadedThreadStatus struct {
	Type string `json:"type"`
}

type IdleThreadStatus struct {
	Type string `json:"type"`
}

type SystemErrorThreadStatus struct {
	Type string `json:"type"`
}

type ActiveThreadStatus struct {
	ActiveFlags []ThreadActiveFlag `json:"activeFlags"`
	Type        string             `json:"type"`
}

type ThreadActiveFlag string

const (
	ThreadActiveFlagWaitingOnApproval  ThreadActiveFlag = "waitingOnApproval"
	ThreadActiveFlagWaitingOnUserInput ThreadActiveFlag = "waitingOnUserInput"
)

type ThreadStatusChangedNotificationEnvelope struct {
	Method string                          `json:"method"`
	Params ThreadStatusChangedNotification `json:"params"`
}

type ThreadStatusChangedNotification struct {
	Status   ThreadStatus `json:"status"`
	ThreadID string       `json:"threadId"`
}

type ThreadArchivedNotificationEnvelope struct {
	Method string                     `json:"method"`
	Params ThreadArchivedNotification `json:"params"`
}

type ThreadArchivedNotification struct {
	ThreadID string `json:"threadId"`
}

type ThreadDeletedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ThreadDeletedNotification `json:"params"`
}

type ThreadDeletedNotification struct {
	ThreadID string `json:"threadId"`
}

type ThreadUnarchivedNotificationEnvelope struct {
	Method string                       `json:"method"`
	Params ThreadUnarchivedNotification `json:"params"`
}

type ThreadUnarchivedNotification struct {
	ThreadID string `json:"threadId"`
}

type ThreadClosedNotificationEnvelope struct {
	Method string                   `json:"method"`
	Params ThreadClosedNotification `json:"params"`
}

type ThreadClosedNotification struct {
	ThreadID string `json:"threadId"`
}

type SkillsChangedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params SkillsChangedNotification `json:"params"`
}

type SkillsChangedNotification struct {
}

type ThreadNameUpdatedNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params ThreadNameUpdatedNotification `json:"params"`
}

type ThreadNameUpdatedNotification struct {
	ThreadID   string  `json:"threadId"`
	ThreadName *string `json:"threadName,omitempty"`
}

type ThreadGoalUpdatedNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params ThreadGoalUpdatedNotification `json:"params"`
}

type ThreadGoalUpdatedNotification struct {
	Goal     ThreadGoal `json:"goal"`
	ThreadID string     `json:"threadId"`
	TurnID   *string    `json:"turnId,omitempty"`
}

type ThreadGoal struct {
	CreatedAt       int64            `json:"createdAt"`
	Objective       string           `json:"objective"`
	Status          ThreadGoalStatus `json:"status"`
	ThreadID        string           `json:"threadId"`
	TimeUsedSeconds int64            `json:"timeUsedSeconds"`
	TokenBudget     *int64           `json:"tokenBudget,omitempty"`
	TokensUsed      int64            `json:"tokensUsed"`
	UpdatedAt       int64            `json:"updatedAt"`
}

type ThreadGoalClearedNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params ThreadGoalClearedNotification `json:"params"`
}

type ThreadGoalClearedNotification struct {
	ThreadID string `json:"threadId"`
}

type ThreadEnvironmentConnectedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params EnvironmentConnectionNotification `json:"params"`
}

type ThreadEnvironmentDisconnectedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params EnvironmentConnectionNotification `json:"params"`
}

type ThreadSettingsUpdatedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params ThreadSettingsUpdatedNotification `json:"params"`
}

type ThreadSettingsUpdatedNotification struct {
	ThreadID       string         `json:"threadId"`
	ThreadSettings ThreadSettings `json:"threadSettings"`
}

type ThreadSettings struct {
	ActivePermissionProfile *ActivePermissionProfile `json:"activePermissionProfile,omitempty"`
	ApprovalPolicy          AskForApproval           `json:"approvalPolicy"`
	ApprovalsReviewer       ApprovalsReviewer        `json:"approvalsReviewer"`
	CollaborationMode       CollaborationMode        `json:"collaborationMode"`
	Cwd                     AbsolutePathBuf          `json:"cwd"`
	Effort                  *ReasoningEffort         `json:"effort,omitempty"`
	Model                   string                   `json:"model"`
	ModelProvider           string                   `json:"modelProvider"`
	Personality             *Personality             `json:"personality,omitempty"`
	SandboxPolicy           SandboxPolicy            `json:"sandboxPolicy"`
	ServiceTier             *string                  `json:"serviceTier,omitempty"`
	Summary                 *ReasoningSummary        `json:"summary,omitempty"`
}

type ThreadTokenUsageUpdatedNotificationEnvelope struct {
	Method string                              `json:"method"`
	Params ThreadTokenUsageUpdatedNotification `json:"params"`
}

type ThreadTokenUsageUpdatedNotification struct {
	ThreadID   string           `json:"threadId"`
	TokenUsage ThreadTokenUsage `json:"tokenUsage"`
	TurnID     string           `json:"turnId"`
}

type ThreadTokenUsage struct {
	Last               TokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64              `json:"modelContextWindow,omitempty"`
	Total              TokenUsageBreakdown `json:"total"`
}

type TurnStartedNotificationEnvelope struct {
	Method string                  `json:"method"`
	Params TurnStartedNotification `json:"params"`
}

type TurnStartedNotification struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

type HookStartedNotificationEnvelope struct {
	Method string                  `json:"method"`
	Params HookStartedNotification `json:"params"`
}

type TurnCompletedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params TurnCompletedNotification `json:"params"`
}

type TurnCompletedNotification struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

type HookCompletedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params HookCompletedNotification `json:"params"`
}

type TurnDiffUpdatedNotificationEnvelope struct {
	Method string                      `json:"method"`
	Params TurnDiffUpdatedNotification `json:"params"`
}

type TurnDiffUpdatedNotification struct {
	Diff     string `json:"diff"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type TurnPlanUpdatedNotificationEnvelope struct {
	Method string                      `json:"method"`
	Params TurnPlanUpdatedNotification `json:"params"`
}

type TurnPlanUpdatedNotification struct {
	Explanation *string        `json:"explanation,omitempty"`
	Plan        []TurnPlanStep `json:"plan"`
	ThreadID    string         `json:"threadId"`
	TurnID      string         `json:"turnId"`
}

type TurnPlanStep struct {
	Status TurnPlanStepStatus `json:"status"`
	Step   string             `json:"step"`
}

type TurnPlanStepStatus string

const (
	TurnPlanStepStatusPending    TurnPlanStepStatus = "pending"
	TurnPlanStepStatusInProgress TurnPlanStepStatus = "inProgress"
	TurnPlanStepStatusCompleted  TurnPlanStepStatus = "completed"
)

type ItemStartedNotificationEnvelope struct {
	Method string                  `json:"method"`
	Params ItemStartedNotification `json:"params"`
}

type ItemAutoApprovalReviewStartedNotificationEnvelope struct {
	Method string                                        `json:"method"`
	Params ItemGuardianApprovalReviewStartedNotification `json:"params"`
}

type ItemAutoApprovalReviewCompletedNotificationEnvelope struct {
	Method string                                          `json:"method"`
	Params ItemGuardianApprovalReviewCompletedNotification `json:"params"`
}

type ItemCompletedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ItemCompletedNotification `json:"params"`
}

type ItemAgentMessageDeltaNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params AgentMessageDeltaNotification `json:"params"`
}

type ItemPlanDeltaNotificationEnvelope struct {
	Method string                `json:"method"`
	Params PlanDeltaNotification `json:"params"`
}

type CommandExecOutputDeltaNotificationEnvelope struct {
	Method string                             `json:"method"`
	Params CommandExecOutputDeltaNotification `json:"params"`
}

type ProcessOutputDeltaNotificationEnvelope struct {
	Method string                         `json:"method"`
	Params ProcessOutputDeltaNotification `json:"params"`
}

type ProcessExitedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ProcessExitedNotification `json:"params"`
}

type ItemCommandExecutionOutputDeltaNotificationEnvelope struct {
	Method string                                  `json:"method"`
	Params CommandExecutionOutputDeltaNotification `json:"params"`
}

type ItemCommandExecutionTerminalInteractionNotificationEnvelope struct {
	Method string                          `json:"method"`
	Params TerminalInteractionNotification `json:"params"`
}

type TerminalInteractionNotification struct {
	ItemID    string `json:"itemId"`
	ProcessID string `json:"processId"`
	Stdin     string `json:"stdin"`
	ThreadID  string `json:"threadId"`
	TurnID    string `json:"turnId"`
}

type ItemFileChangeOutputDeltaNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params FileChangeOutputDeltaNotification `json:"params"`
}

type ItemFileChangePatchUpdatedNotificationEnvelope struct {
	Method string                             `json:"method"`
	Params FileChangePatchUpdatedNotification `json:"params"`
}

type ServerRequestResolvedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params ServerRequestResolvedNotification `json:"params"`
}

type ServerRequestResolvedNotification struct {
	RequestID RequestId `json:"requestId"`
	ThreadID  string    `json:"threadId"`
}

type ItemMcpToolCallProgressNotificationEnvelope struct {
	Method string                          `json:"method"`
	Params McpToolCallProgressNotification `json:"params"`
}

type McpServerOauthLoginCompletedNotificationEnvelope struct {
	Method string                                   `json:"method"`
	Params McpServerOauthLoginCompletedNotification `json:"params"`
}

type McpServerStartupStatusUpdatedNotificationEnvelope struct {
	Method string                             `json:"method"`
	Params McpServerStatusUpdatedNotification `json:"params"`
}

type AccountUpdatedNotificationEnvelope struct {
	Method string                     `json:"method"`
	Params AccountUpdatedNotification `json:"params"`
}

type AccountRateLimitsUpdatedNotificationEnvelope struct {
	Method string                               `json:"method"`
	Params AccountRateLimitsUpdatedNotification `json:"params"`
}

type AppListUpdatedNotificationEnvelope struct {
	Method string                     `json:"method"`
	Params AppListUpdatedNotification `json:"params"`
}

type RemoteControlStatusChangedNotificationEnvelope struct {
	Method string                                 `json:"method"`
	Params RemoteControlStatusChangedNotification `json:"params"`
}

type ExternalAgentConfigImportProgressNotificationEnvelope struct {
	Method string                                        `json:"method"`
	Params ExternalAgentConfigImportProgressNotification `json:"params"`
}

type ExternalAgentConfigImportCompletedNotificationEnvelope struct {
	Method string                                         `json:"method"`
	Params ExternalAgentConfigImportCompletedNotification `json:"params"`
}

type FsChangedNotificationEnvelope struct {
	Method string                `json:"method"`
	Params FsChangedNotification `json:"params"`
}

type ItemReasoningSummaryTextDeltaNotificationEnvelope struct {
	Method string                                `json:"method"`
	Params ReasoningSummaryTextDeltaNotification `json:"params"`
}

type ItemReasoningSummaryPartAddedNotificationEnvelope struct {
	Method string                                `json:"method"`
	Params ReasoningSummaryPartAddedNotification `json:"params"`
}

type ItemReasoningTextDeltaNotificationEnvelope struct {
	Method string                         `json:"method"`
	Params ReasoningTextDeltaNotification `json:"params"`
}

type ThreadCompactedNotificationEnvelope struct {
	Method string                       `json:"method"`
	Params ContextCompactedNotification `json:"params"`
}

type ModelReroutedNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ModelReroutedNotification `json:"params"`
}

type ModelVerificationNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params ModelVerificationNotification `json:"params"`
}

type TurnModerationMetadataNotificationEnvelope struct {
	Method string                             `json:"method"`
	Params TurnModerationMetadataNotification `json:"params"`
}

type TurnModerationMetadataNotification struct {
	Metadata any    `json:"metadata"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type ModelSafetyBufferingUpdatedNotificationEnvelope struct {
	Method string                                  `json:"method"`
	Params ModelSafetyBufferingUpdatedNotification `json:"params"`
}

type WarningNotificationEnvelope struct {
	Method string              `json:"method"`
	Params WarningNotification `json:"params"`
}

type WarningNotification struct {
	Message  string  `json:"message"`
	ThreadID *string `json:"threadId,omitempty"`
}

type GuardianWarningNotificationEnvelope struct {
	Method string                      `json:"method"`
	Params GuardianWarningNotification `json:"params"`
}

type DeprecationNoticeNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params DeprecationNoticeNotification `json:"params"`
}

type ConfigWarningNotificationEnvelope struct {
	Method string                    `json:"method"`
	Params ConfigWarningNotification `json:"params"`
}

type FuzzyFileSearchSessionUpdatedNotificationEnvelope struct {
	Method string                                    `json:"method"`
	Params FuzzyFileSearchSessionUpdatedNotification `json:"params"`
}

type FuzzyFileSearchSessionCompletedNotificationEnvelope struct {
	Method string                                      `json:"method"`
	Params FuzzyFileSearchSessionCompletedNotification `json:"params"`
}

type ThreadRealtimeStartedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params ThreadRealtimeStartedNotification `json:"params"`
}

type ThreadRealtimeStartedNotification struct {
	RealtimeSessionID *string                     `json:"realtimeSessionId,omitempty"`
	ThreadID          string                      `json:"threadId"`
	Version           RealtimeConversationVersion `json:"version"`
}

type ThreadRealtimeItemAddedNotificationEnvelope struct {
	Method string                              `json:"method"`
	Params ThreadRealtimeItemAddedNotification `json:"params"`
}

type ThreadRealtimeItemAddedNotification struct {
	Item     any    `json:"item"`
	ThreadID string `json:"threadId"`
}

type ThreadRealtimeTranscriptDeltaNotificationEnvelope struct {
	Method string                                    `json:"method"`
	Params ThreadRealtimeTranscriptDeltaNotification `json:"params"`
}

type ThreadRealtimeTranscriptDeltaNotification struct {
	Delta    string `json:"delta"`
	Role     string `json:"role"`
	ThreadID string `json:"threadId"`
}

type ThreadRealtimeTranscriptDoneNotificationEnvelope struct {
	Method string                                   `json:"method"`
	Params ThreadRealtimeTranscriptDoneNotification `json:"params"`
}

type ThreadRealtimeTranscriptDoneNotification struct {
	Role     string `json:"role"`
	Text     string `json:"text"`
	ThreadID string `json:"threadId"`
}

type ThreadRealtimeOutputAudioDeltaNotificationEnvelope struct {
	Method string                                     `json:"method"`
	Params ThreadRealtimeOutputAudioDeltaNotification `json:"params"`
}

type ThreadRealtimeOutputAudioDeltaNotification struct {
	Audio    ThreadRealtimeAudioChunk `json:"audio"`
	ThreadID string                   `json:"threadId"`
}

type ThreadRealtimeAudioChunk struct {
	Data              string  `json:"data"`
	ItemID            *string `json:"itemId,omitempty"`
	NumChannels       int64   `json:"numChannels"`
	SampleRate        int64   `json:"sampleRate"`
	SamplesPerChannel *int64  `json:"samplesPerChannel,omitempty"`
}

type ThreadRealtimeSdpNotificationEnvelope struct {
	Method string                        `json:"method"`
	Params ThreadRealtimeSdpNotification `json:"params"`
}

type ThreadRealtimeSdpNotification struct {
	Sdp      string `json:"sdp"`
	ThreadID string `json:"threadId"`
}

type ThreadRealtimeErrorNotificationEnvelope struct {
	Method string                          `json:"method"`
	Params ThreadRealtimeErrorNotification `json:"params"`
}

type ThreadRealtimeErrorNotification struct {
	Message  string `json:"message"`
	ThreadID string `json:"threadId"`
}

type ThreadRealtimeClosedNotificationEnvelope struct {
	Method string                           `json:"method"`
	Params ThreadRealtimeClosedNotification `json:"params"`
}

type ThreadRealtimeClosedNotification struct {
	Reason   *string `json:"reason,omitempty"`
	ThreadID string  `json:"threadId"`
}

type WindowsWorldWritableWarningNotificationEnvelope struct {
	Method string                                  `json:"method"`
	Params WindowsWorldWritableWarningNotification `json:"params"`
}

type WindowsWorldWritableWarningNotification struct {
	ExtraCount  int64    `json:"extraCount"`
	FailedScan  bool     `json:"failedScan"`
	SamplePaths []string `json:"samplePaths"`
}

type WindowsSandboxSetupCompletedNotificationEnvelope struct {
	Method string                                   `json:"method"`
	Params WindowsSandboxSetupCompletedNotification `json:"params"`
}

type WindowsSandboxSetupCompletedNotification struct {
	Error   *string                 `json:"error,omitempty"`
	Mode    WindowsSandboxSetupMode `json:"mode"`
	Success bool                    `json:"success"`
}

type AccountLoginCompletedNotificationEnvelope struct {
	Method string                            `json:"method"`
	Params AccountLoginCompletedNotification `json:"params"`
}

type ServerRequest struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ServerRequest) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "method") {
		if disc, ok := unionDiscriminator(data, "method"); ok {
			switch disc {
			case "item/commandExecution/requestApproval":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ItemCommandExecutionRequestApprovalRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/fileChange/requestApproval":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ItemFileChangeRequestApprovalRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/tool/requestUserInput":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ItemToolRequestUserInputRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "mcpServer/elicitation/request":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v McpServerElicitationRequestRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/permissions/requestApproval":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ItemPermissionsRequestApprovalRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "item/tool/call":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ItemToolCallRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "account/chatgptAuthTokens/refresh":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AccountChatgptAuthTokensRefreshRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "attestation/generate":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v AttestationGenerateRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "applyPatchApproval":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ApplyPatchApprovalRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "execCommandApproval":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params")) {
					break
				}
				var v ExecCommandApprovalRequestEnvelope
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ItemCommandExecutionRequestApprovalRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ItemFileChangeRequestApprovalRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ItemToolRequestUserInputRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v McpServerElicitationRequestRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ItemPermissionsRequestApprovalRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ItemToolCallRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AccountChatgptAuthTokensRefreshRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v AttestationGenerateRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ApplyPatchApprovalRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "id", "method", "params") {
		var v ExecCommandApprovalRequestEnvelope
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ServerRequest) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type ItemCommandExecutionRequestApprovalRequestEnvelope struct {
	ID     RequestId                             `json:"id"`
	Method string                                `json:"method"`
	Params CommandExecutionRequestApprovalParams `json:"params"`
}

type ItemFileChangeRequestApprovalRequestEnvelope struct {
	ID     RequestId                       `json:"id"`
	Method string                          `json:"method"`
	Params FileChangeRequestApprovalParams `json:"params"`
}

type ItemToolRequestUserInputRequestEnvelope struct {
	ID     RequestId                  `json:"id"`
	Method string                     `json:"method"`
	Params ToolRequestUserInputParams `json:"params"`
}

type ToolRequestUserInputParams struct {
	AutoResolutionMs *int64                         `json:"autoResolutionMs,omitempty"`
	IsBlocking       bool                           `json:"isBlocking"`
	ItemID           string                         `json:"itemId"`
	Questions        []ToolRequestUserInputQuestion `json:"questions"`
	ThreadID         string                         `json:"threadId"`
	TurnID           string                         `json:"turnId"`
}

type ToolRequestUserInputQuestion struct {
	Header   string                       `json:"header"`
	ID       string                       `json:"id"`
	IsOther  *bool                        `json:"isOther,omitempty"`
	IsSecret *bool                        `json:"isSecret,omitempty"`
	Options  []ToolRequestUserInputOption `json:"options,omitempty"`
	Question string                       `json:"question"`
}

type ToolRequestUserInputOption struct {
	Description string `json:"description"`
	Label       string `json:"label"`
}

type McpServerElicitationRequestRequestEnvelope struct {
	ID     RequestId                         `json:"id"`
	Method string                            `json:"method"`
	Params McpServerElicitationRequestParams `json:"params"`
}

type ItemPermissionsRequestApprovalRequestEnvelope struct {
	ID     RequestId                        `json:"id"`
	Method string                           `json:"method"`
	Params PermissionsRequestApprovalParams `json:"params"`
}

type ItemToolCallRequestEnvelope struct {
	ID     RequestId             `json:"id"`
	Method string                `json:"method"`
	Params DynamicToolCallParams `json:"params"`
}

type AccountChatgptAuthTokensRefreshRequestEnvelope struct {
	ID     RequestId                      `json:"id"`
	Method string                         `json:"method"`
	Params ChatgptAuthTokensRefreshParams `json:"params"`
}

type AttestationGenerateRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params AttestationGenerateParams `json:"params"`
}

type ApplyPatchApprovalRequestEnvelope struct {
	ID     RequestId                `json:"id"`
	Method string                   `json:"method"`
	Params ApplyPatchApprovalParams `json:"params"`
}

type ExecCommandApprovalRequestEnvelope struct {
	ID     RequestId                 `json:"id"`
	Method string                    `json:"method"`
	Params ExecCommandApprovalParams `json:"params"`
}

type SkillDependencies struct {
	Tools []SkillToolDependency `json:"tools"`
}

type SkillToolDependency struct {
	Command     *string `json:"command,omitempty"`
	Description *string `json:"description,omitempty"`
	Transport   *string `json:"transport,omitempty"`
	Type        string  `json:"type"`
	Url         *string `json:"url,omitempty"`
	Value       string  `json:"value"`
}

type SkillErrorInfo struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

type SkillMetadata struct {
	Dependencies     *SkillDependencies `json:"dependencies,omitempty"`
	Description      string             `json:"description"`
	Enabled          bool               `json:"enabled"`
	Interface        *SkillInterface    `json:"interface,omitempty"`
	Name             string             `json:"name"`
	Path             AbsolutePathBuf    `json:"path"`
	Scope            SkillScope         `json:"scope"`
	ShortDescription *string            `json:"shortDescription,omitempty"`
}

type SkillScope string

const (
	SkillScopeUser   SkillScope = "user"
	SkillScopeRepo   SkillScope = "repo"
	SkillScopeSystem SkillScope = "system"
	SkillScopeAdmin  SkillScope = "admin"
)

type SkillsConfigWriteResponse struct {
	EffectiveEnabled bool `json:"effectiveEnabled"`
}

type SkillsExtraRootsSetResponse struct {
}

type SkillsListEntry struct {
	Cwd    string           `json:"cwd"`
	Errors []SkillErrorInfo `json:"errors"`
	Skills []SkillMetadata  `json:"skills"`
}

type SkillsListResponse struct {
	Data []SkillsListEntry `json:"data"`
}

type ThreadApproveGuardianDeniedActionResponse struct {
}

type ThreadArchiveResponse struct {
}

type ThreadCompactStartResponse struct {
}

type ThreadDeleteResponse struct {
}

type ThreadExtra struct {
}

type ThreadForkResponse struct {
	ApprovalPolicy     AskForApproval        `json:"approvalPolicy"`
	ApprovalsReviewer  ApprovalsReviewer     `json:"approvalsReviewer"`
	Cwd                AbsolutePathBuf       `json:"cwd"`
	InstructionSources []LegacyAppPathString `json:"instructionSources,omitempty"`
	Model              string                `json:"model"`
	ModelProvider      string                `json:"modelProvider"`
	ReasoningEffort    *ReasoningEffort      `json:"reasoningEffort,omitempty"`
	Sandbox            SandboxPolicy         `json:"sandbox"`
	ServiceTier        *string               `json:"serviceTier,omitempty"`
	Thread             Thread                `json:"thread"`
}

type ThreadGoalClearResponse struct {
	Cleared bool `json:"cleared"`
}

type ThreadGoalGetResponse struct {
	Goal *ThreadGoal `json:"goal,omitempty"`
}

type ThreadGoalSetResponse struct {
	Goal ThreadGoal `json:"goal"`
}

type ThreadHistoryMode string

const (
	ThreadHistoryModeLegacy    ThreadHistoryMode = "legacy"
	ThreadHistoryModePaginated ThreadHistoryMode = "paginated"
)

type ThreadInjectItemsResponse struct {
}

type ThreadItemEntry struct {
	Item   ThreadItem `json:"item"`
	TurnID string     `json:"turnId"`
}

type ThreadListResponse struct {
	BackwardsCursor *string  `json:"backwardsCursor,omitempty"`
	Data            []Thread `json:"data"`
	NextCursor      *string  `json:"nextCursor,omitempty"`
}

type ThreadLoadedListResponse struct {
	Data       []string `json:"data"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

type ThreadMemoryMode string

const (
	ThreadMemoryModeEnabled  ThreadMemoryMode = "enabled"
	ThreadMemoryModeDisabled ThreadMemoryMode = "disabled"
)

type ThreadMetadataUpdateResponse struct {
	Thread Thread `json:"thread"`
}

type ThreadReadResponse struct {
	Thread Thread `json:"thread"`
}

type ThreadRealtimeInitialItem struct {
	Role ConversationTextRole `json:"role"`
	Text string               `json:"text"`
}

type ThreadRealtimeStartTransport struct {
	Value any             `json:"-"`
	Raw   json.RawMessage `json:"-"`
}

func (u *ThreadRealtimeStartTransport) UnmarshalJSON(data []byte) error {
	if u == nil {
		return nil
	}
	u.Raw = append(json.RawMessage(nil), data...)
	if jsonHasField(data, "type") {
		if disc, ok := unionDiscriminator(data, "type"); ok {
			switch disc {
			case "websocket":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "type")) {
					break
				}
				var v WebsocketThreadRealtimeStartTransport
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			case "webrtc":
				if !(jsonLooksObject(data) && jsonHasKeys(data, "sdp", "type")) {
					break
				}
				var v WebrtcThreadRealtimeStartTransport
				if err := json.Unmarshal(data, &v); err != nil {
					return err
				}
				u.Value = &v
				return nil
			}
		}
		u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "type") {
		var v WebsocketThreadRealtimeStartTransport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	if jsonLooksObject(data) && jsonHasKeys(data, "sdp", "type") {
		var v WebrtcThreadRealtimeStartTransport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = &v
		return nil
	}
	u.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}
	return nil
}

func (u ThreadRealtimeStartTransport) MarshalJSON() ([]byte, error) {
	if u.Value == nil && len(u.Raw) != 0 {
		return append(json.RawMessage(nil), u.Raw...), nil
	}
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

type WebsocketThreadRealtimeStartTransport struct {
	Type string `json:"type"`
}

type WebrtcThreadRealtimeStartTransport struct {
	Sdp  string `json:"sdp"`
	Type string `json:"type"`
}

type ThreadResumeInitialTurnsPageParams struct {
	ItemsView     *TurnItemsView `json:"itemsView,omitempty"`
	Limit         *int64         `json:"limit,omitempty"`
	SortDirection *SortDirection `json:"sortDirection,omitempty"`
}

type ThreadResumeResponse struct {
	ApprovalPolicy     AskForApproval        `json:"approvalPolicy"`
	ApprovalsReviewer  ApprovalsReviewer     `json:"approvalsReviewer"`
	Cwd                AbsolutePathBuf       `json:"cwd"`
	InstructionSources []LegacyAppPathString `json:"instructionSources,omitempty"`
	Model              string                `json:"model"`
	ModelProvider      string                `json:"modelProvider"`
	ReasoningEffort    *ReasoningEffort      `json:"reasoningEffort,omitempty"`
	Sandbox            SandboxPolicy         `json:"sandbox"`
	ServiceTier        *string               `json:"serviceTier,omitempty"`
	Thread             Thread                `json:"thread"`
}

type ThreadRollbackResponse struct {
	Thread Thread `json:"thread"`
}

type ThreadSearchResult struct {
	Snippet string `json:"snippet"`
	Thread  Thread `json:"thread"`
}

type ThreadSearchSortKey string

const (
	ThreadSearchSortKeyCreatedAt ThreadSearchSortKey = "created_at"
	ThreadSearchSortKeyUpdatedAt ThreadSearchSortKey = "updated_at"
	ThreadSearchSortKeyRecencyAt ThreadSearchSortKey = "recency_at"
)

type ThreadSectionCreateResponse struct {
	Section ThreadSection `json:"section"`
}

type ThreadSectionDeleteResponse struct {
}

type ThreadSectionListResponse struct {
	Data       []ThreadSection `json:"data"`
	NextCursor *string         `json:"nextCursor,omitempty"`
}

type ThreadSectionMoveResponse struct {
}

type ThreadSectionUpdateResponse struct {
	Section ThreadSection `json:"section"`
}

type ThreadSetNameResponse struct {
}

type ThreadShellCommandResponse struct {
}

type ThreadStartResponse struct {
	ApprovalPolicy     AskForApproval        `json:"approvalPolicy"`
	ApprovalsReviewer  ApprovalsReviewer     `json:"approvalsReviewer"`
	Cwd                AbsolutePathBuf       `json:"cwd"`
	InstructionSources []LegacyAppPathString `json:"instructionSources,omitempty"`
	Model              string                `json:"model"`
	ModelProvider      string                `json:"modelProvider"`
	ReasoningEffort    *ReasoningEffort      `json:"reasoningEffort,omitempty"`
	Sandbox            SandboxPolicy         `json:"sandbox"`
	ServiceTier        *string               `json:"serviceTier,omitempty"`
	Thread             Thread                `json:"thread"`
}

type ThreadUnarchiveResponse struct {
	Thread Thread `json:"thread"`
}

type ThreadUnsubscribeResponse struct {
	Status ThreadUnsubscribeStatus `json:"status"`
}

type ThreadUnsubscribeStatus string

const (
	ThreadUnsubscribeStatusNotLoaded     ThreadUnsubscribeStatus = "notLoaded"
	ThreadUnsubscribeStatusNotSubscribed ThreadUnsubscribeStatus = "notSubscribed"
	ThreadUnsubscribeStatusUnsubscribed  ThreadUnsubscribeStatus = "unsubscribed"
)

type ToolRequestUserInputAnswer struct {
	Answers []string `json:"answers"`
}

type ToolRequestUserInputResponse struct {
	Answers map[string]ToolRequestUserInputAnswer `json:"answers"`
}

type TurnEnvironmentParams struct {
	Cwd                   LegacyAppPathString   `json:"cwd"`
	EnvironmentID         string                `json:"environmentId"`
	RuntimeWorkspaceRoots []LegacyAppPathString `json:"runtimeWorkspaceRoots,omitempty"`
}

type TurnInterruptResponse struct {
}

type TurnStartResponse struct {
	Turn Turn `json:"turn"`
}

type TurnSteerResponse struct {
	TurnID string `json:"turnId"`
}

type TurnsPage struct {
	BackwardsCursor *string `json:"backwardsCursor,omitempty"`
	Data            []Turn  `json:"data"`
	NextCursor      *string `json:"nextCursor,omitempty"`
}

type WindowsSandboxReadiness string

const (
	WindowsSandboxReadinessReady          WindowsSandboxReadiness = "ready"
	WindowsSandboxReadinessNotConfigured  WindowsSandboxReadiness = "notConfigured"
	WindowsSandboxReadinessUpdateRequired WindowsSandboxReadiness = "updateRequired"
)

type WindowsSandboxReadinessResponse struct {
	Status WindowsSandboxReadiness `json:"status"`
}

type WindowsSandboxSetupStartResponse struct {
	Started bool `json:"started"`
}

type CodexAppServerProtocolSchemas struct {
}

type CodexAppServerProtocolV2Schemas struct {
}

type V2 any
