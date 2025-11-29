package transport

import (
	"errors"
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

func (c *GRPCClient) getConn(target string, opts []string) (*grpc.ClientConn, error) {
	dialOpts := c.parseDialOptions(opts)
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		c.log.Error("GRPC connection create error", zap.Error(err))
		return conn, err
	}
	return conn, nil
}

func (c *GRPCClient) createCtx(cfgMd map[string]string) context.Context {
	ctx := context.Background()
	if len(cfgMd) > 0 {
		md := metadata.New(cfgMd)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

func (c *GRPCClient) parseEndpoint(endpoint string) (string, string, error) {
	parts := strings.Split(endpoint, "/")
	if len(parts) != 2 {
		c.log.Error("Cfg endpoint is invalid",
			zap.String("endpoint", endpoint))
		return "", "", errors.New("Invalid endpoint")
	}
	return parts[0], parts[1], nil
}

func (c *GRPCClient) getFilesSVC(sName string, fds []*desc.FileDescriptor) *desc.ServiceDescriptor {
	var svc *desc.ServiceDescriptor
	for _, fd := range fds {
		svc = fd.FindService(sName)
		if svc != nil {
			return svc
		}
	}
	return nil
}

func (c *GRPCClient) parseOutput(dMsg *dynamic.Message) []byte {
	if jsonBytes, err := dMsg.MarshalJSON(); err == nil {
		return jsonBytes
	} else if err != nil {
		c.log.Error("Marshal response error", zap.Error(err))
	}

	if textBytes, err := dMsg.MarshalText(); err == nil {
		return textBytes
	} else if err != nil {
		c.log.Error("Couldn't format the response into text", zap.Error(err))
	}

	if binaryBytes, err := dMsg.Marshal(); err == nil {
		return binaryBytes
	} else if err != nil {
		c.log.Error("Couldn't format the response into bytes", zap.Error(err))
	}

	c.log.Error("All marshaling attempts failed")
	return nil
}
