package gcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

// gcpAcmeEabResource google cloud platform, get eab account for acme client
type gcpAcmeEabResource struct {
	credentialsJSON []byte
}

func NewGcpAcmeEabResource() resource.Resource {
	return &gcpAcmeEabResource{}
}

func (r *gcpAcmeEabResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acme_eab"
}

func (r *gcpAcmeEabResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"eab_account_expires_days": &schema.Int64Attribute{ // default is 60 days
				Required: false,
				Optional: true,
			},
			"name": &schema.StringAttribute{
				Computed: true,
			},
			"key_id": &schema.StringAttribute{
				Computed: true,
			},
			"hmac_base64": &schema.StringAttribute{
				Computed: true,
			},
			"create_at": &schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *gcpAcmeEabResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		// this data available on apply stage
		return
	}
	j, ok := req.ProviderData.([]byte)
	if !ok {
		resp.Diagnostics.AddError("credentials_json format error", "")
		return
	}
	r.credentialsJSON = j
}

type resourceConfigFormat struct {
	EabAccountExpiresDays int64        `tfsdk:"eab_account_expires_days"`
	Name                  types.String `tfsdk:"name"`
	KeyID                 types.String `tfsdk:"key_id"`
	HmacBase64            types.String `tfsdk:"hmac_base64"`
	CreateAt              types.Int64  `tfsdk:"create_at"` // the unix timestamp of create eab account
}

func (r *gcpAcmeEabResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var cfg resourceConfigFormat
	d := req.Plan.Get(ctx, &cfg)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Create req.Config.Get error")
		return
	}
	eabResp, err := gcpGetEab(ctx, r.credentialsJSON, nil)
	if err != nil {
		tflog.Error(ctx, "gcpGetEab error", map[string]interface{}{
			"error":            err.Error(),
			"credentials_json": string(r.credentialsJSON),
		})
		resp.Diagnostics.AddError("gcpGetEab error", err.Error())
		return
	}
	cfg.Name = basetypes.NewStringValue(eabResp.Name)
	cfg.KeyID = basetypes.NewStringValue(eabResp.KeyId)
	cfg.HmacBase64 = basetypes.NewStringValue(eabResp.B64MacKey)
	cfg.CreateAt = basetypes.NewInt64Value(time.Now().Unix())
	resp.State.Set(ctx, &cfg)
}

func (r *gcpAcmeEabResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var cfg resourceConfigFormat
	d := req.State.Get(ctx, &cfg)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read req.State.Get error")
		return
	}
	eabData := externalAccountKeyResp{
		Name:      cfg.Name.String(),
		KeyId:     cfg.KeyID.String(),
		B64MacKey: cfg.HmacBase64.String(),
	}
	if len(eabData.Name) == 0 || len(eabData.KeyId) == 0 || len(eabData.B64MacKey) == 0 {
		tflog.Info(ctx, "account not create")
		return
	}
	eabResp, err := gcpGetEab(ctx, r.credentialsJSON, &eabData)
	if err != nil {
		tflog.Error(ctx, "gcpGetEab error", map[string]interface{}{
			"error":            err.Error(),
			"credentials_json": string(r.credentialsJSON),
		})
		resp.Diagnostics.AddError("gcpGetEab error", err.Error())
		return
	}
	if cfg.Name.String() == eabResp.Name &&
		cfg.KeyID.String() == eabResp.KeyId &&
		cfg.HmacBase64.String() == eabResp.B64MacKey {
		tflog.Info(ctx, "account not change")
		return
	}
	cfg.CreateAt = basetypes.NewInt64Value(time.Now().Unix())
	cfg.Name = basetypes.NewStringValue(eabResp.Name)
	cfg.KeyID = basetypes.NewStringValue(eabResp.KeyId)
	cfg.HmacBase64 = basetypes.NewStringValue(eabResp.B64MacKey)
	resp.State.Set(ctx, &cfg)
}

const defaultEabAccountExpiresDays = 60

func (r *gcpAcmeEabResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var cfg resourceConfigFormat
	d := req.State.Get(ctx, &cfg)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update req.State.Get error")
		return
	}
	if cfg.EabAccountExpiresDays <= 0 {
		cfg.EabAccountExpiresDays = defaultEabAccountExpiresDays
	}
	createAt := cfg.CreateAt.ValueInt64()
	if int64(time.Since(time.Unix(createAt, 0)).Hours()/24) < cfg.EabAccountExpiresDays {
		tflog.Info(ctx, "eab account not expires")
		var cfgReq resourceConfigFormat
		d := req.Config.Get(ctx, &cfgReq)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "Update req.Config.Get error")
			return
		}
		cfg.EabAccountExpiresDays = cfgReq.EabAccountExpiresDays // must set to config value
		// or error happend like this:
		// When applying changes to st-gcp_acme_eab.eab, provider "provider[\"registry.terraform.io/myklst/st-gcp\"]"
		// produced an unexpected new value: .eab_account_expires_days: was cty.NumberIntVal(2), but now cty.NumberIntVal(1).
		resp.State.Set(ctx, &cfg)
		return
	}
	eabData := externalAccountKeyResp{
		Name:      cfg.Name.String(),
		KeyId:     cfg.KeyID.String(),
		B64MacKey: cfg.HmacBase64.String(),
	}
	eabResp, err := gcpGetEab(ctx, r.credentialsJSON, &eabData)
	if err != nil {
		tflog.Error(ctx, "gcpGetEab error", map[string]interface{}{
			"error":            err.Error(),
			"credentials_json": string(r.credentialsJSON),
		})
		resp.Diagnostics.AddError("gcpGetEab error", err.Error())
		return
	}
	cfg.Name = basetypes.NewStringValue(eabResp.Name)
	cfg.KeyID = basetypes.NewStringValue(eabResp.KeyId)
	cfg.HmacBase64 = basetypes.NewStringValue(eabResp.B64MacKey)
	cfg.CreateAt = basetypes.NewInt64Value(time.Now().Unix())
	resp.State.Set(ctx, &cfg)
}

func (r *gcpAcmeEabResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

}

type gcloudCred struct {
	Type                    string `json:"type"`
	ProjectId               string `json:"project_id"`
	PrivateKeyId            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientId                string `json:"client_id"`
	AuthUri                 string `json:"auth_uri"`
	TokenUri                string `json:"token_uri"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl       string `json:"client_x509_cert_url"`
}

type externalAccountKeyResp struct {
	Name      string `json:"name"`
	KeyId     string `json:"keyId"`
	B64MacKey string `json:"b64MacKey"`
}

// gcpGetEab create a eab account
// see: https://cloud.google.com/certificate-manager/docs/reference/public-ca/rest/v1/projects.locations.externalAccountKeys/create
func gcpGetEab(ctx context.Context, credentialsJSON []byte, old *externalAccountKeyResp) (*externalAccountKeyResp, error) {
	ctx = tflog.NewSubsystem(ctx, "gcpGetEab")
	cred := &gcloudCred{}
	if err := json.Unmarshal(credentialsJSON, &cred); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Google private key: %v", err)
	}
	url := "https://www.googleapis.com/auth/cloud-platform"
	conf, err := google.JWTConfigFromJSON(credentialsJSON, url)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT config: %v", err)
	}
	var api = fmt.Sprintf("https://publicca.googleapis.com/v1beta1/projects/%s/locations/global/externalAccountKeys", cred.ProjectId)
	var resp *http.Response
	if old != nil {
		old.B64MacKey = base64.StdEncoding.Strict().EncodeToString([]byte(old.B64MacKey))
		buf, _ := json.Marshal(old)
		resp, err = conf.Client(context.Background()).Post(api, "application/json", bytes.NewReader(buf))
	} else {
		resp, err = conf.Client(context.Background()).Post(api, "application/json", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to request Google Public CA API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("url:" + api + ", error:" + string(body))
	}
	var eab externalAccountKeyResp
	if err := json.Unmarshal(body, &eab); err != nil {
		return nil, fmt.Errorf("failed to unmarshal EAB response: %v", err)
	}
	eabMacKey, err := base64.StdEncoding.DecodeString(eab.B64MacKey)
	if err != nil {
		return nil, fmt.Errorf("failed to Base64 decode EAB MacKey: %v", err)
	}
	eab.B64MacKey = string(eabMacKey)
	return &eab, nil
}
