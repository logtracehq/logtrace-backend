package awsses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"gitlab.com/logtrace/logtrace/config"
	email "gitlab.com/logtrace/logtrace/internal/pkg/email"
)

type Client struct {
	svc *sesv2.Client
}

func New(cfg *config.Config) (*Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Email.SES.Region),
	}

	if cfg.Email.SES.AccessKey != "" && cfg.Email.SES.SecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.Email.SES.AccessKey, cfg.Email.SES.SecretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	return &Client{svc: sesv2.NewFromConfig(awsCfg)}, nil
}

func (c *Client) Send(ctx context.Context, opts email.SendOptions) (string, error) {
	if err := opts.Validate(); err != nil {
		return "", err
	}

	input := &sesv2.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{opts.Recipient.String()},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Charset: aws.String("UTF-8"),
						Data:    aws.String(opts.HTML),
					},
				},
				Subject: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(opts.Subject),
				},
			},
		},
		FromEmailAddress: aws.String(opts.Sender.String()),
	}

	out, err := c.svc.SendEmail(ctx, input)
	if err != nil {
		return "", fmt.Errorf("sending email via SES: %w", err)
	}

	return aws.ToString(out.MessageId), nil
}

func (c *Client) Close() error {
	return nil
}
