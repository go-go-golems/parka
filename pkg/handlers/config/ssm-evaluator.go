package config

import (
	"context"
	"os"

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

	var configOpts []func(*config.LoadOptions) error
	if envRegion != "" {
		configOpts = append(configOpts, config.WithRegion(envRegion))
	} else {
		configOpts = append(configOpts, config.WithRegion("us-east-1"))
	}

	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load AWS SDK config")
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
