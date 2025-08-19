package transport

import (
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	dialOpts := parseDialOptions(cfg.DialOptions)
	conn, err := grpc.NewClient(cfg.Target, dialOpts...)
	if err != nil {
		log.Printf("GRCP connection create error: %v", err)
		return Result{}, err
	}
	defer conn.Close()
	return Result{}, err
}
