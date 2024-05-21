package redis

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/stordto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/logger"
	"github.com/go-redis/redis/v8"
)

var ErrStorVersionMismatch = errors.New("error version mismatch")

const (
	chCheckPointKeyTemplate = "%s-ch-checkpoint-%s"
)

type Storage struct {
	log             glog.Logger
	client          redis.UniversalClient
	dbPrefix        string
	channelName     string
	chCheckPointKey string
}

func NewStorage(
	ctx context.Context,
	addrs []string,
	password string,
	withTLS bool,
	rootCAs []string,
	dbPrefix string,
	channelName string,
) (*Storage, error) {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentStorage,
			ChName:    channelName,
		}.Fields()...)

	redisOpts := &redis.UniversalOptions{
		Addrs:    addrs,
		Password: password,
		ReadOnly: false,
	}

	if withTLS {
		certPool := x509.NewCertPool()

		for _, rootCA := range rootCAs {
			cert, err := os.ReadFile(rootCA)
			if err != nil {
				return nil,
					errorshlp.WrapWithDetails(fmt.Errorf("failed to read root CA certificate %s: %w", rootCA, err),
						nerrors.ErrTypeInternal,
						nerrors.ComponentStorage)
			}

			if ok := certPool.AppendCertsFromPEM(cert); !ok {
				return nil,
					errorshlp.WrapWithDetails(fmt.Errorf(
						"failed to add root CA certificate %s to the certificate pool", rootCA),
						nerrors.ErrTypeInternal,
						nerrors.ComponentStorage)
			}
		}

		redisOpts.TLSConfig = &tls.Config{RootCAs: certPool}
	}

	return &Storage{
		log:             log,
		client:          redis.NewUniversalClient(redisOpts),
		dbPrefix:        dbPrefix,
		channelName:     channelName,
		chCheckPointKey: fmt.Sprintf(chCheckPointKeyTemplate, dbPrefix, channelName),
	}, nil
}

func (stor *Storage) SaveCheckPoints(ctx context.Context, cp *stordto.ChCheckPoint) (*stordto.ChCheckPoint, error) {
	newCp := *cp

	err := stor.client.Watch(ctx, func(tx *redis.Tx) error {
		current, ok, err := stor.getChCheckPoint(ctx, tx)
		if err != nil {
			return err
		}

		if ok {
			if current.Ver != cp.Ver {
				return ErrStorVersionMismatch
			}
			newCp.Ver++
		}

		_, err = tx.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			return stor.setChCheckPoint(ctx, pipe, &newCp)
		})

		return err
	}, stor.chCheckPointKey)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(
			err, nerrors.ErrTypeRedis,
			nerrors.ComponentStorage)
	}

	return &newCp, nil
}

func (stor *Storage) LoadCheckPoints(ctx context.Context) (*stordto.ChCheckPoint, bool, error) {
	res, ok, err := stor.getChCheckPoint(ctx, stor.client)
	return res, ok, errorshlp.WrapWithDetails(
		err, nerrors.ErrTypeRedis,
		nerrors.ComponentStorage)
}

func (stor *Storage) setChCheckPoint(ctx context.Context, cl redis.Cmdable, cp *stordto.ChCheckPoint) error {
	data, err := encodeData(cp)
	if err != nil {
		return err
	}
	cl.Set(ctx, stor.chCheckPointKey, data, 0)
	return nil
}

func (stor *Storage) getChCheckPoint(ctx context.Context, cl redis.Cmdable) (*stordto.ChCheckPoint, bool, error) {
	data, err := cl.Get(ctx, stor.chCheckPointKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	res := &stordto.ChCheckPoint{}
	if err := decodeData(data, res); err != nil {
		return nil, false, err
	}

	// if SrcCollectFromBlockNums is not deserialized (data stored in another format),
	// consider the get checkpoint operation as incorrect and return false
	if res.SrcCollectFromBlockNums == nil {
		return res, false, nil
	}

	return res, true, nil
}

func (stor *Storage) RemoveAllData(ctx context.Context) error {
	res := stor.client.Del(ctx, stor.chCheckPointKey)
	return res.Err()
}

func encodeData(data interface{}) ([]byte, error) {
	var bytebuffer bytes.Buffer
	e := gob.NewEncoder(&bytebuffer)
	if err := e.Encode(data); err != nil {
		return nil,
			errorshlp.WrapWithDetails(
				err, nerrors.ErrTypeParsing,
				nerrors.ComponentStorage)
	}
	return bytebuffer.Bytes(), nil
}

func decodeData(data []byte, res interface{}) error {
	bytebuffer := bytes.NewBuffer(data)
	d := gob.NewDecoder(bytebuffer)
	if err := d.Decode(res); err != nil {
		return errorshlp.WrapWithDetails(
			err, nerrors.ErrTypeParsing,
			nerrors.ComponentStorage)
	}
	return nil
}
