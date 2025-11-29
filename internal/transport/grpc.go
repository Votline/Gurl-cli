package transport

import (
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	refl "github.com/jhump/protoreflect/grpcreflect"

	"Gurl-cli/internal/config"
)


func (c *GRPCClient) parseDialOptions(opts []string) []grpc.DialOption {
	dialOpts := make([]grpc.DialOption, 0)
	for _, opt := range opts {
		switch opt {
		case "insecure":
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(
				insecure.NewCredentials()))
		case "block":
			dialOpts = append(dialOpts, grpc.WithBlock())
		default:
			c.log.Error("Unsupported dial option\nInsecure will be used", zap.Any("options", opts))
			continue
		}
	}
	if len(dialOpts) == 0 {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(
			insecure.NewCredentials()))
	}
	return dialOpts
}

func (c *GRPCClient) GRPC(cfg *config.GRPCConfig) (Result, error) {
	if len(cfg.ProtoFiles) == 0 {
		return c.grpcReflect(cfg)
	} else {
		return c.grpcFiles(cfg)
	}
}

func (c *GRPCClient) grpcReflect(cfg *config.GRPCConfig) (Result, error) {
	c.log.Error("Reflect is used")
	conn, err := c.getConn(cfg.Target, cfg.DialOptions)
	if err != nil {
		c.log.Error("Create gRPC connection error", zap.Error(err))
		return Result{}, err
	}
	defer conn.Close()

	ctx := c.createCtx(cfg.Metadata)
	rc := refl.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))

	serviceName, methodName, err := c.parseEndpoint(cfg.Endpoint)
	if err != nil {
		c.log.Error("Parse endpoint err", zap.Error(err))
		return Result{}, err
	}

	svc, err := rc.ResolveService(serviceName)
	if err != nil {
		c.log.Error("Service not found",
			zap.String("service", serviceName),
			zap.Error(err))
		return Result{}, err
	}
	method := svc.FindMethodByName(methodName)
	if method == nil {
		c.log.Error("Method not found in service",
			zap.String("method", method.String()),
			zap.String("service", svc.String()),
			zap.Error(err))
		return Result{}, err
	}
	
	msg := dynamic.NewMessage(method.GetInputType())
	if err := msg.UnmarshalJSON(cfg.Data); err != nil {
		c.log.Error("Protobuf to JSON error", zap.Error(err))
		return Result{}, err
	}
	
	stub := grpcdynamic.NewStub(conn)
	resp, err := stub.InvokeRpc(ctx, method, msg)
	if err != nil {
		textMsg, _ := msg.MarshalText()
		c.log.Error("Response getting error",
			zap.String("message", string(textMsg)),
			zap.Error(err))
		return Result{}, err
	}

	dMsg, ok := resp.(*dynamic.Message)
		if !ok {
			c.log.Error("There is an response, but it was not possible to convert it to a dynamic Message")
			return Result{}, err
		}

	out := c.parseOutput(dMsg)
	parsed := make(map[string]any)
	if err := json.Unmarshal(out, &parsed); err != nil {
		c.log.Error("Couldn't unmarshall the output into json", zap.Error(err))
		return Result{JSON: nil, RawBody: out}, nil
	}

	return Result{JSON: parsed, RawBody: out}, nil
}

func (c *GRPCClient) grpcFiles(cfg *config.GRPCConfig) (Result, error) {
	c.log.Info(".proto files is used")
	conn, err := c.getConn(cfg.Target, cfg.DialOptions)
	if err != nil {
		c.log.Error("Create gRPC connection error", zap.Error(err))
		return Result{}, err
	}
	defer conn.Close()
	ctx := c.createCtx(cfg.Metadata)
	stub := grpcdynamic.NewStub(conn)

	parser := protoparse.Parser{}
	fds, err := parser.ParseFiles(cfg.ProtoFiles...)
	if err != nil {
		c.log.Error("Couldn't parse .proto files", zap.Error(err))
	}

	serviceName, serviceMethod, err := c.parseEndpoint(cfg.Endpoint)
	if err != nil {
		c.log.Error("Parse endpoint error", zap.Error(err))
		return Result{}, err
	}
	
	svc := c.getFilesSVC(serviceName, fds)
	if svc == nil {
		c.log.Error("Couldn't find the service",
			zap.String("service name", serviceName))
		return Result{}, err
	}

	method := svc.FindMethodByName(serviceMethod)
	if method == nil {
		c.log.Error("Couldn't find the service method",
			zap.String("service name", serviceMethod))
		return Result{}, err
	}

	req := dynamic.NewMessage(method.GetInputType())
	if err := req.UnmarshalJSON(cfg.Data); err != nil {
		c.log.Error("Unmarshal request error", zap.Error(err))
		return Result{}, err
	}

	resp, err := stub.InvokeRpc(ctx, method, req)
	if err != nil {
		c.log.Warn("Invoke rpc error", zap.Error(err))
		return Result{}, err
	}

	dMsg, ok := resp.(*dynamic.Message)
	if !ok {
		c.log.Error("There is an response, but it was not possible to convert it to a dynamic Message")
		return Result{}, err
	}

	out := c.parseOutput(dMsg)
	parsed := make(map[string]any)
	if err := json.Unmarshal(out, &parsed); err != nil {
		c.log.Error("Couldn't unmarshall the output into json err")
		return Result{JSON: nil, RawBody: out}, nil
	}

	return Result{JSON: parsed, RawBody: out}, nil
}
