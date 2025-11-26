package core

import (
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"Gurl-cli/internal/config"
	"Gurl-cli/internal/transport"
	gen "Gurl-cli/internal/generate"
)

type core struct{
	log *zap.Logger
	cfgPath string
	ic bool

	parser *config.Parser
	client *transport.HTTPClient
}

func prettyPrint(data any, indent string) {
	switch v := any(data).(type) {
	case map[string]any:
		fmt.Println(indent+"[")
		for key, val := range v {
			switch valTyped := val.(type) {
			case map[string]any, []any:
				fmt.Printf("%s    %s:", indent, key)
				prettyPrint(valTyped, indent + "    ")
			default:
				fmt.Printf("%s    %s: %v", indent, key, valTyped)
			}
		}
		fmt.Println(indent+"]")
	case []any:
		for _, elem := range v {
			prettyPrint(elem, indent+"    ")
		}
	default:
		fmt.Printf("%s%v", indent, v)
	}
}

func (c *core) handleHTTP(cfg *config.HTTPConfig) {
	var err error
	var res transport.Result
	switch cfg.Method {
	case "GET":
		res, err = c.client.Get(cfg)
	case "POST":
		res, err = c.client.Post(cfg)
	case "PUT":
		res, err = c.client.Put(cfg)
	case "DELETE":
		res, err = c.client.Del(cfg)
	}
	if err != nil {
		c.log.Fatal("Make http request error",
			zap.String("Method", cfg.Method),
			zap.Error(err))
	}

	fmt.Println(res.Raw.Status)
	if res.JSON != nil {
		prettyPrint(res.JSON, "    ")
	} else if res.RawBody != nil {
		fmt.Println(string(res.RawBody))
	} else {
		fmt.Println("Empty response body")
	}

	cfg.SetResponse(string(res.RawBody))
	c.parser.ConfigUpd(cfg, c.cfgPath)
}

func (c *core) handleGRPC(cfg *config.GRPCConfig) {
	res, err := transport.GRPC(cfg)
	if err != nil {
		c.log.Fatal("Make gRPC request error", zap.Error(err))
	}

	if res.JSON != nil {
		prettyPrint(res.JSON, "    ")
	} else if res.RawBody != nil {
		fmt.Println(string(res.RawBody))
	} else {
		fmt.Println("Empty response body")
	}

	cfg.SetResponse(string(res.RawBody))
	c.parser.ConfigUpd(cfg, c.cfgPath)
}

func (c *core) handleRequest() {
	cfgs, err := c.parser.Decode(c.cfgPath)
	if err != nil {
		c.log.Fatal("Error when trying to get the config", zap.Error(err))
	}

	for idx, cfg := range cfgs {
		parsed, err := c.parser.Parsing(cfg, cfgs)
		if err != nil {
			c.log.Error("Parse error", zap.Error(err))
			continue
		}
		strIdx := strconv.Itoa(idx+1)
		parsed.SetID(strIdx)
		cfgs[idx] = parsed

		switch v := cfg.(type) {
		case *config.HTTPConfig:
			c.handleHTTP(v)
		case *config.GRPCConfig:
			c.handleGRPC(v)
		default:
			c.log.Error("Invalid config type", zap.Any("type", v))
		}
	}
}

func Start(cfgType, cfgPath string, cfgCreate, ic bool, ckPath string, log *zap.Logger) {
	c := &core{
		log: log,
		cfgPath: cfgPath,
		ic: ic,
		parser: config.NewParser(log),
		client: transport.NewClient(ic, ckPath, log),
	}
	if !cfgCreate {
		switch cfgType {
		case "http", "grpc", "repeated":
			c.handleRequest()
			return
		default:
			log.Fatal("Invalid config type", zap.String("config type", cfgType))
		}
	}
	if err := gen.InitConfig(cfgPath, cfgType); err != nil {
		log.Fatal("Generate config error", zap.Error(err))
	}
}
