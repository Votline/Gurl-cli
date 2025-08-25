package transport

import (
	"log"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/jhump/protoreflect/dynamic"
	refl "github.com/jhump/protoreflect/grpcreflect"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"

	"Gurl-cli/internal/config"
)


func parseDialOptions(opts []string) []grpc.DialOption {
	dialOpts := make([]grpc.DialOption, 0)
	for _, opt := range opts {
		switch opt {
		case "insecure":
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(
				insecure.NewCredentials()))
		case "block":
			dialOpts = append(dialOpts, grpc.WithBlock())
		default:
			log.Printf("Unsupported dial option: %s\nInsecure will be used",opt)
			continue
		}
	}
	if len(dialOpts) == 0 {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(
			insecure.NewCredentials()))
	}
	return dialOpts
}

func GRPC(cfg *config.GRPCConfig) (Result, error) {
	conn, err := getConn(cfg.Target, cfg.DialOptions)
	if err != nil {
		log.Printf("Get connection err: %v", err)
		return Result{}, err
	}
	defer conn.Close()

	ctx := createCtx(cfg.Metadata)
	rc := refl.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))

	serviceName, methodName, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		log.Printf("Parse endpoint err: %v", err)
		return Result{}, err
	}

	svc, err := rc.ResolveService(serviceName)
	if err != nil {
		log.Printf("Service %s not found: %v", serviceName, err)
		return Result{}, err
	}
	method := svc.FindMethodByName(methodName)
	if method == nil {
		log.Printf("Method %s not found in service %s.\nerror: %v",
			method, svc, err)
		return Result{}, err
	}
	
	msg := dynamic.NewMessage(method.GetInputType())
	if err := msg.UnmarshalJSON(cfg.Data); err != nil {
		log.Printf("Protobuf to JSON error: %v", err)
		return Result{}, err
	}
	
	stub := grpcdynamic.NewStub(conn)
	resp, err := stub.InvokeRpc(ctx, method, msg)
	if err != nil {
		textMsg, _ := msg.MarshalText()
		log.Printf("Response getting error: %v\n%s", err, textMsg)
		return Result{}, err
	}

	dynamicMsg, ok := resp.(*dynamic.Message)
	if !ok {
		log.Printf("Invalid response type: %v", err)
		return Result{}, err
	}

	out := parseOutput(dynamicMsg)
	rawOut, _ := dynamicMsg.MarshalText()

	parsed := make(map[string]any)
	if err := json.Unmarshal(out, &parsed); err != nil {
		log.Printf("Couldn't unmarshall the output into json: %v", err)
		return Result{JSON: nil, RawBody: out}, nil
	}

	return Result{JSON: parsed, RawBody: rawOut}, nil
}
