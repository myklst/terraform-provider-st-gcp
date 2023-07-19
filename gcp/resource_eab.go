package gcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
}

func NewGcpAcmeEabResource() resource.Resource {
	return &gcpAcmeEabResource{}
}

func (r *gcpAcmeEabResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_eab"
}

func (r *gcpAcmeEabResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"credentials_json": &schema.StringAttribute{
				Required:  true,
				Optional:  false,
				Sensitive: true,
			},
			"key_id": &schema.StringAttribute{
				Computed: true,
			},
			"hmac_base64": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *gcpAcmeEabResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

type resourceConfigFormat struct {
	CredentialsJSON string       `tfsdk:"credentials_json"`
	KeyID           types.String `tfsdk:"key_id"`
	HmacBase64      types.String `tfsdk:"hmac_base64"`
}

func (r *gcpAcmeEabResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var cfg resourceConfigFormat
	d := req.Plan.Get(ctx, &cfg)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "gcpAcmeEabResource req.Config.Get error")
		return
	}
	keyID, b64MacKey, err := gcpGetEab(cfg.CredentialsJSON)
	if err != nil {
		tflog.Error(ctx, "gcpGetEab error", map[string]interface{}{
			"error":            err.Error(),
			"credentials_json": cfg.CredentialsJSON,
		})
		resp.Diagnostics.AddError("gcpGetEab error", err.Error())
		return
	}
	cfg.KeyID = basetypes.NewStringValue(keyID)
	cfg.HmacBase64 = basetypes.NewStringValue(b64MacKey)
	resp.State.Set(ctx, &cfg)
}

func (r *gcpAcmeEabResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

func (r *gcpAcmeEabResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

func gcpGetEab(credentialsJSON string) (string, string, error) {
	cred := &gcloudCred{}
	if err := json.Unmarshal([]byte(credentialsJSON), &cred); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal Google private key: %v", err)
	}
	url := "https://www.googleapis.com/auth/cloud-platform"
	conf, err := google.JWTConfigFromJSON([]byte(credentialsJSON), url)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate JWT config: %v", err)
	}
	var api = fmt.Sprintf("https://publicca.googleapis.com/v1beta1/projects/%s/locations/global/externalAccountKeys", cred.ProjectId)
	resp, err := conf.Client(context.Background()).Post(api, "application/json", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to request Google Public CA API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("url:" + api + ", error:" + string(body))
	}
	var eab externalAccountKeyResp
	if err := json.Unmarshal(body, &eab); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal EAB response: %v", err)
	}
	eabMacKey, err := base64.StdEncoding.DecodeString(eab.B64MacKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to Base64 decode EAB MacKey: %v", err)
	}
	return eab.KeyId, string(eabMacKey), nil
}
