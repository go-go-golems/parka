package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type SsmEvaluator struct {
	client *ssm.Client
	ctx    context.Context
}

func NewSsmEvaluator(ctx context.Context) (*SsmEvaluator, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load AWS SDK config")
	}

	return &SsmEvaluator{
		client: ssm.NewFromConfig(cfg),
		ctx:    ctx,
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
					return err
				})
				log.Info().Msgf("getting parameter %s from AWS SSM", k)
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
