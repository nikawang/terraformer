// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package iot

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
	"github.com/aws/aws-sdk-go-v2/private/protocol"
	"github.com/aws/aws-sdk-go-v2/private/protocol/restjson"
)

// The input for the EnableTopicRuleRequest operation.
type EnableTopicRuleInput struct {
	_ struct{} `type:"structure"`

	// The name of the topic rule to enable.
	//
	// RuleName is a required field
	RuleName *string `location:"uri" locationName:"ruleName" min:"1" type:"string" required:"true"`
}

// String returns the string representation
func (s EnableTopicRuleInput) String() string {
	return awsutil.Prettify(s)
}

// Validate inspects the fields of the type to determine if they are valid.
func (s *EnableTopicRuleInput) Validate() error {
	invalidParams := aws.ErrInvalidParams{Context: "EnableTopicRuleInput"}

	if s.RuleName == nil {
		invalidParams.Add(aws.NewErrParamRequired("RuleName"))
	}
	if s.RuleName != nil && len(*s.RuleName) < 1 {
		invalidParams.Add(aws.NewErrParamMinLen("RuleName", 1))
	}

	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s EnableTopicRuleInput) MarshalFields(e protocol.FieldEncoder) error {
	e.SetValue(protocol.HeaderTarget, "Content-Type", protocol.StringValue("application/json"), protocol.Metadata{})

	if s.RuleName != nil {
		v := *s.RuleName

		metadata := protocol.Metadata{}
		e.SetValue(protocol.PathTarget, "ruleName", protocol.QuotedValue{ValueMarshaler: protocol.StringValue(v)}, metadata)
	}
	return nil
}

type EnableTopicRuleOutput struct {
	_ struct{} `type:"structure"`
}

// String returns the string representation
func (s EnableTopicRuleOutput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s EnableTopicRuleOutput) MarshalFields(e protocol.FieldEncoder) error {
	return nil
}

const opEnableTopicRule = "EnableTopicRule"

// EnableTopicRuleRequest returns a request value for making API operation for
// AWS IoT.
//
// Enables the rule.
//
//    // Example sending a request using EnableTopicRuleRequest.
//    req := client.EnableTopicRuleRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
func (c *Client) EnableTopicRuleRequest(input *EnableTopicRuleInput) EnableTopicRuleRequest {
	op := &aws.Operation{
		Name:       opEnableTopicRule,
		HTTPMethod: "POST",
		HTTPPath:   "/rules/{ruleName}/enable",
	}

	if input == nil {
		input = &EnableTopicRuleInput{}
	}

	req := c.newRequest(op, input, &EnableTopicRuleOutput{})
	req.Handlers.Unmarshal.Remove(restjson.UnmarshalHandler)
	req.Handlers.Unmarshal.PushBackNamed(protocol.UnmarshalDiscardBodyHandler)
	return EnableTopicRuleRequest{Request: req, Input: input, Copy: c.EnableTopicRuleRequest}
}

// EnableTopicRuleRequest is the request type for the
// EnableTopicRule API operation.
type EnableTopicRuleRequest struct {
	*aws.Request
	Input *EnableTopicRuleInput
	Copy  func(*EnableTopicRuleInput) EnableTopicRuleRequest
}

// Send marshals and sends the EnableTopicRule API request.
func (r EnableTopicRuleRequest) Send(ctx context.Context) (*EnableTopicRuleResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &EnableTopicRuleResponse{
		EnableTopicRuleOutput: r.Request.Data.(*EnableTopicRuleOutput),
		response:              &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// EnableTopicRuleResponse is the response type for the
// EnableTopicRule API operation.
type EnableTopicRuleResponse struct {
	*EnableTopicRuleOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// EnableTopicRule request.
func (r *EnableTopicRuleResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}