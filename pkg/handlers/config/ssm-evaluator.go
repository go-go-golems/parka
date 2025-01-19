package config

import (
	"context"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type SsmEvaluator struct {
	client    *ssm.Client
	stsClient *sts.Client
	ctx       context.Context
}

func NewSsmEvaluator(ctx context.Context) (*SsmEvaluator, error) {
	// Check environment for region
	envRegion := os.Getenv("AWS_REGION")
	if envRegion == "" {
		envRegion = os.Getenv("AWS_DEFAULT_REGION")
	}

	log.Debug().
		Str("AWS_REGION", os.Getenv("AWS_REGION")).
		Str("AWS_DEFAULT_REGION", os.Getenv("AWS_DEFAULT_REGION")).
		Msg("AWS region environment variables")

	var configOpts []func(*config.LoadOptions) error
	if envRegion != "" {
		log.Debug().Str("region", envRegion).Msg("Using region from environment")
		configOpts = append(configOpts, config.WithRegion(envRegion))
	} else {
		log.Debug().Msg("No region found in environment, defaulting to us-east-1")
		configOpts = append(configOpts, config.WithRegion("us-east-1"))
	}

	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load AWS SDK config")
	}

	// Get credentials for logging
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("failed to retrieve AWS credentials for debug logging")
	} else {
		// Only show first 4 chars of access key
		truncatedKey := creds.AccessKeyID
		if len(truncatedKey) > 4 {
			truncatedKey = truncatedKey[:4] + strings.Repeat("*", len(truncatedKey)-4)
		}
		log.Debug().
			Str("access_key", truncatedKey).
			Str("provider", string(creds.Source)).
			Msg("AWS credentials loaded")
	}

	log.Debug().
		Str("region", cfg.Region).
		Str("retry_mode", string(cfg.RetryMode)).
		Msg("AWS config loaded")

	// Get caller identity for additional context
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Warn().Err(err).Msg("failed to get AWS caller identity")
	} else {
		log.Debug().
			Str("account", *identity.Account).
			Str("arn", *identity.Arn).
			Str("user_id", *identity.UserId).
			Msg("AWS caller identity")
	}

	return &SsmEvaluator{
		client:    ssm.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
		ctx:       ctx,
	}, nil
}

func (s *SsmEvaluator) Evaluate(node interface{}) (interface{}, bool, error) {
	switch value := node.(type) {
	case map[string]interface{}:
		if len(value) == 1 && value["_aws_ssm"] != nil {
			if ssmKey, ok := value["_aws_ssm"]; ok {
				v, err := EvaluateConfigEntry(ssmKey)
				if err != nil {
					return nil, false, errors.Wrap(err, "failed to evaluate SSM key")
				}
				k, ok := v.(string)
				if !ok {
					return nil, false, errors.New("'_aws_ssm' key must have a string value")
				}
				eg, ctx := errgroup.WithContext(s.ctx)
				var result *ssm.GetParameterOutput
				eg.Go(func() error {
					var err error
					result, err = s.client.GetParameter(ctx, &ssm.GetParameterInput{
						Name:           aws.String(k),
						WithDecryption: aws.Bool(true),
					})

					if err != nil {
						// Get current identity for error context
						identity, identityErr := s.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
						if identityErr != nil {
							log.Info().
								Str("parameter", k).
								Str("error", identityErr.Error()).
								Msg("failed to get SSM parameter - current AWS identity")
						} else {
							log.Info().
								Str("parameter", k).
								Str("account", *identity.Account).
								Str("arn", *identity.Arn).
								Msg("failed to get SSM parameter - current AWS identity")
						}
					}
					return err
				})
				log.Info().Msgf("getting parameter %s from AWS SSM", k)
				log.Info().Msgf("result: %+v", result)
				if err := eg.Wait(); err != nil {
					return nil, false, errors.Wrap(err, "failed to get parameter from AWS SSM")
				}

				return *result.Parameter.Value, true, nil
			}
		}

		return nil, false, nil
	default:
		return nil, false, nil
	}
}
