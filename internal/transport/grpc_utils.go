package transport

import (
	"log"
	"errors"
	"context"
	"strings"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"github.com/jhump/protoreflect/dynamic"
)

func getConn(target string, opts []string) (*grpc.ClientConn, error) {
	dialOpts := parseDialOptions(opts)
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		log.Printf("GRPC connection create error: %v", err)
		return conn, err
	}
	return conn, nil
}

func createCtx(cfgMd map[string]string) context.Context {
	ctx := context.Background()
	if len(cfgMd) > 0 {
		md := metadata.New(cfgMd)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

func parseEndpoint(endpoint string) (string, string, error) {
	parts := strings.Split(endpoint, "/")
	if len(parts) != 2 {
		log.Printf("Cfg endpoint is invalid: %s", endpoint)
		return "", "", errors.New("Invalid endpoint")
	}
	return parts[0], parts[1], nil
}

func parseOutput(dMsg *dynamic.Message) []byte {
	out, err := dMsg.MarshalJSON()
	if err != nil {
		log.Printf("Marshal response error: %v", err)
		rawOut, err := dMsg.MarshalText()
		if err != nil {
			log.Printf("Couldn't format the response into text: %v", err)
			rawBytes, err := dMsg.Marshal()
			if err != nil {
				log.Printf("Couldn't format the response into bytes: %v", err)
				return nil
			}
			return rawBytes
		}
		return rawOut
	}
	return out
}
