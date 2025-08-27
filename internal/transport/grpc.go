package transport

import (
	"log"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/desc/protoparse"
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
	if len(cfg.ProtoFiles) == 0 {
		return grpcReflect(cfg)
	} else {
		return grpcFiles(cfg)
	}
}

func grpcReflect(cfg *config.GRPCConfig) (Result, error) {
	log.Printf("Reflect is used")
	conn, err := getConn(cfg.Target, cfg.DialOptions)
	if err != nil {
		log.Printf("Create gRPC connection error: %v", err)
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

	dMsg, ok := resp.(*dynamic.Message)
		if !ok {
			log.Printf("There is an response, but it was not possible to convert it to a dynamic Message")
			return Result{}, err
		}

	out := parseOutput(dMsg)
	parsed := make(map[string]any)
	if err := json.Unmarshal(out, &parsed); err != nil {
		log.Printf("Couldn't unmarshall the output into json: %v", err)
		return Result{JSON: nil, RawBody: out}, nil
	}

	return Result{JSON: parsed, RawBody: out}, nil
}

func grpcFiles(cfg *config.GRPCConfig) (Result, error) {
	log.Printf(".proto files is used")
	conn, err := getConn(cfg.Target, cfg.DialOptions)
	if err != nil {
		log.Printf("Create gRPC connection error: %v", err)
		return Result{}, err
	}
	defer conn.Close()
	ctx := createCtx(cfg.Metadata)
	stub := grpcdynamic.NewStub(conn)

	parser := protoparse.Parser{}
	fds, err := parser.ParseFiles(cfg.ProtoFiles...)
	if err != nil {
		log.Printf("Couldn't parse .proto files: %v", )
	}

	serviceName, serviceMethod, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		log.Printf("Parse endpoint error: %v", err)
		return Result{}, err
	}
	
	svc := getFilesSVC(serviceName, fds)
	if svc == nil {
		log.Printf("Couldn't find the service: %s", serviceName)
		return Result{}, err
	}

	method := svc.FindMethodByName(serviceMethod)
	if method == nil {
		log.Printf("Couldn't find the service method: %s", serviceMethod)
		return Result{}, err
	}

	req := dynamic.NewMessage(method.GetInputType())
	if err := req.UnmarshalJSON(cfg.Data); err != nil {
		log.Printf("Unmarshal request error: %v", err)
		return Result{}, err
	}

	resp, err := stub.InvokeRpc(ctx, method, req)
	if err != nil {
		log.Printf("Couldn't invoke the rpc method: %v", err)
		return Result{}, err
	}

	dMsg, ok := resp.(*dynamic.Message)
	if !ok {
		log.Printf("There is an response, but it was not possible to convert it to a dynamic Message")
		return Result{}, err
	}

	out := parseOutput(dMsg)
	parsed := make(map[string]any)
	if err := json.Unmarshal(out, &parsed); err != nil {
		log.Printf("Couldn't unmarshall the output into json: %v err")
		return Result{JSON: nil, RawBody: out}, nil
	}

	return Result{JSON: parsed, RawBody: out}, nil
}
