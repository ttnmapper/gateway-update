// Package Openapi provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.8.1 DO NOT EDIT.
package Openapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

const (
	OAuth2Scopes = "OAuth2.Scopes"
)

// ContactInfo defines model for ContactInfo.
type ContactInfo struct {
	Email *openapi_types.Email `json:"email,omitempty"`
	Name  *string              `json:"name,omitempty"`
	Url   *string              `json:"url,omitempty"`
}

// Error defines model for Error.
type Error struct {
	Message *string `json:"message,omitempty"`
}

// Frequency (Hz)
type Frequency int64

// FrequencyPlan defines model for FrequencyPlan.
type FrequencyPlan struct {

	// Frequency (Hz)
	FskChannel           *Frequency             `json:"fskChannel,omitempty"`
	LoraMultiSFChannels  *[]Frequency           `json:"loraMultiSFChannels,omitempty"`
	LoraSingleSFChannels *[]LoRaSingleSFChannel `json:"loraSingleSFChannels,omitempty"`
	Region               interface{}            `json:"region"`
}

// Gateway defines model for Gateway.
type Gateway struct {
	// Embedded struct due to allOf(#/components/schemas/GatewayIdentifier)
	GatewayIdentifier `yaml:",inline"`
	// Embedded fields due to inline allOf schema
	AntennaCount     *int         `json:"antennaCount,omitempty"`
	AntennaPlacement *interface{} `json:"antennaPlacement,omitempty"`

	// Cluster that the gateway is connected to
	ClusterID *string `json:"clusterID,omitempty"`

	// 64-bit EUI (hex)
	Eui *string `json:"eui,omitempty"`

	// Whether the gateway produces fine timestamps
	FineTimestamps *bool     `json:"fineTimestamps,omitempty"`
	Location       *Location `json:"location,omitempty"`

	// Whether the gateway is currently online
	Online *bool `json:"online,omitempty"`
}

// GatewayDetails defines model for GatewayDetails.
type GatewayDetails struct {
	// Embedded struct due to allOf(#/components/schemas/Gateway)
	Gateway `yaml:",inline"`
	// Embedded fields due to inline allOf schema
	AdministrativeContact *ContactInfo   `json:"administrativeContact,omitempty"`
	FrequencyPlan         *FrequencyPlan `json:"frequencyPlan,omitempty"`

	// Received message rate (messages per second)
	RxRate           *float32     `json:"rxRate,omitempty"`
	TechnicalContact *ContactInfo `json:"technicalContact,omitempty"`

	// Transmitted message rate (messages per second)
	TxRate *float32 `json:"txRate,omitempty"`
}

// GatewayIdentifier defines model for GatewayIdentifier.
type GatewayIdentifier struct {

	// Gateway ID
	Id string `json:"id"`

	// LoRa Alliance NetID (hex)
	NetID string `json:"netID"`

	// Tenant within the NetID
	TenantID *string `json:"tenantID,omitempty"`
}

// LoRaSingleSFChannel defines model for LoRaSingleSFChannel.
type LoRaSingleSFChannel struct {

	// Bandwidth (Hz)
	Bandwidth int `json:"bandwidth"`

	// Frequency (Hz)
	Frequency       Frequency `json:"frequency"`
	SpreadingFactor int       `json:"spreadingFactor"`
}

// Location defines model for Location.
type Location struct {
	// Embedded struct due to allOf(#/components/schemas/PointZ)
	PointZ `yaml:",inline"`
	// Embedded fields due to inline allOf schema

	// Horizontal dilution of precision
	Hdop *float64 `json:"hdop,omitempty"`
}

// Point defines model for Point.
type Point struct {

	// The North–South position (degrees; -90 to +90), where 0 is the equator, North pole is positive, South pole is negative
	Latitude float64 `json:"latitude"`

	// The East-West position (degrees; -180 to +180), where 0 is the Prime Meridian (Greenwich), East is positive, West is negative
	Longitude float64 `json:"longitude"`
}

// PointZ defines model for PointZ.
type PointZ struct {
	// Embedded struct due to allOf(#/components/schemas/Point)
	Point `yaml:",inline"`
	// Embedded fields due to inline allOf schema

	// Altitude (meters), where 0 is the mean sea level
	Altitude *float64 `json:"altitude,omitempty"`
}

// ListGatewaysParams defines parameters for ListGateways.
type ListGatewaysParams struct {

	// Filter by distance from a coordinate
	DistanceWithin *struct {
		// Embedded struct due to allOf(#/components/schemas/Point)
		Point `yaml:",inline"`
		// Embedded fields due to inline allOf schema

		// Distance (meters)
		Distance  float64 `json:"distance"`
		Latitude  float32 `json:"latitude"`
		Longitude float32 `json:"longitude"`
	} `json:"distanceWithin,omitempty"`

	// Number of gateways to skip
	Offset *int `json:"offset,omitempty"`

	// Number of gateways to return. A zero value uses the server's default limit
	Limit *int `json:"limit,omitempty"`

	// Return gateways that were updated since the given timestamp
	UpdatedSince *time.Time `json:"updatedSince,omitempty"`

	// When true, return only gateways that are online
	Online *bool `json:"online,omitempty"`
}

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetGateway request
	GetGateway(ctx context.Context, id GatewayIdentifier, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ListGateways request
	ListGateways(ctx context.Context, params *ListGatewaysParams, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetGateway(ctx context.Context, id GatewayIdentifier, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetGatewayRequest(c.Server, id)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ListGateways(ctx context.Context, params *ListGatewaysParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewListGatewaysRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetGatewayRequest generates requests for GetGateway
func NewGetGatewayRequest(server string, id GatewayIdentifier) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", true, "id", runtime.ParamLocationPath, id)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/gateway/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}
	operationURL := url.URL{
		Path: operationPath,
	}

	queryURL := serverURL.ResolveReference(&operationURL)

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewListGatewaysRequest generates requests for ListGateways
func NewListGatewaysRequest(server string, params *ListGatewaysParams) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/gateways")
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}
	operationURL := url.URL{
		Path: operationPath,
	}

	queryURL := serverURL.ResolveReference(&operationURL)

	queryValues := queryURL.Query()

	if params.DistanceWithin != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("deepObject", true, "distanceWithin", runtime.ParamLocationQuery, *params.DistanceWithin); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.Offset != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "offset", runtime.ParamLocationQuery, *params.Offset); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.Limit != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "limit", runtime.ParamLocationQuery, *params.Limit); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.UpdatedSince != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "updatedSince", runtime.ParamLocationQuery, *params.UpdatedSince); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.Online != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "online", runtime.ParamLocationQuery, *params.Online); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	queryURL.RawQuery = queryValues.Encode()

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetGateway request
	GetGatewayWithResponse(ctx context.Context, id GatewayIdentifier, reqEditors ...RequestEditorFn) (*GetGatewayResponse, error)

	// ListGateways request
	ListGatewaysWithResponse(ctx context.Context, params *ListGatewaysParams, reqEditors ...RequestEditorFn) (*ListGatewaysResponse, error)
}

type GetGatewayResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *GatewayDetails
	JSON400      *Error
}

// Status returns HTTPResponse.Status
func (r GetGatewayResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetGatewayResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ListGatewaysResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *[]Gateway
	JSON400      *Error
}

// Status returns HTTPResponse.Status
func (r ListGatewaysResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ListGatewaysResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetGatewayWithResponse request returning *GetGatewayResponse
func (c *ClientWithResponses) GetGatewayWithResponse(ctx context.Context, id GatewayIdentifier, reqEditors ...RequestEditorFn) (*GetGatewayResponse, error) {
	rsp, err := c.GetGateway(ctx, id, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetGatewayResponse(rsp)
}

// ListGatewaysWithResponse request returning *ListGatewaysResponse
func (c *ClientWithResponses) ListGatewaysWithResponse(ctx context.Context, params *ListGatewaysParams, reqEditors ...RequestEditorFn) (*ListGatewaysResponse, error) {
	rsp, err := c.ListGateways(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseListGatewaysResponse(rsp)
}

// ParseGetGatewayResponse parses an HTTP response from a GetGatewayWithResponse call
func ParseGetGatewayResponse(rsp *http.Response) (*GetGatewayResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer rsp.Body.Close()
	if err != nil {
		return nil, err
	}

	response := &GetGatewayResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest GatewayDetails
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	}

	return response, nil
}

// ParseListGatewaysResponse parses an HTTP response from a ListGatewaysWithResponse call
func ParseListGatewaysResponse(rsp *http.Response) (*ListGatewaysResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer rsp.Body.Close()
	if err != nil {
		return nil, err
	}

	response := &ListGatewaysResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest []Gateway
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	}

	return response, nil
}

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /gateway/{id})
	GetGateway(ctx echo.Context, id GatewayIdentifier) error

	// (GET /gateways)
	ListGateways(ctx echo.Context, params ListGatewaysParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetGateway converts echo context to params.
func (w *ServerInterfaceWrapper) GetGateway(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "id" -------------
	var id GatewayIdentifier

	err = runtime.BindStyledParameterWithLocation("simple", true, "id", runtime.ParamLocationPath, ctx.Param("id"), &id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter id: %s", err))
	}

	ctx.Set(OAuth2Scopes, []string{"networks"})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetGateway(ctx, id)
	return err
}

// ListGateways converts echo context to params.
func (w *ServerInterfaceWrapper) ListGateways(ctx echo.Context) error {
	var err error

	ctx.Set(OAuth2Scopes, []string{"networks"})

	// Parameter object where we will unmarshal all parameters from the context
	var params ListGatewaysParams
	// ------------- Optional query parameter "distanceWithin" -------------

	err = runtime.BindQueryParameter("deepObject", true, false, "distanceWithin", ctx.QueryParams(), &params.DistanceWithin)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter distanceWithin: %s", err))
	}

	// ------------- Optional query parameter "offset" -------------

	err = runtime.BindQueryParameter("form", true, false, "offset", ctx.QueryParams(), &params.Offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter offset: %s", err))
	}

	// ------------- Optional query parameter "limit" -------------

	err = runtime.BindQueryParameter("form", true, false, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// ------------- Optional query parameter "updatedSince" -------------

	err = runtime.BindQueryParameter("form", true, false, "updatedSince", ctx.QueryParams(), &params.UpdatedSince)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter updatedSince: %s", err))
	}

	// ------------- Optional query parameter "online" -------------

	err = runtime.BindQueryParameter("form", true, false, "online", ctx.QueryParams(), &params.Online)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter online: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.ListGateways(ctx, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET(baseURL+"/gateway/:id", wrapper.GetGateway)
	router.GET(baseURL+"/gateways", wrapper.ListGateways)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"GVlRuWxnMW88tQxO/h8AAP//ux5exq4eAAA=",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	var res = make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
