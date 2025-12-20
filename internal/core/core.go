package core

import (
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"Gurl-cli/internal/config"
	gen "Gurl-cli/internal/generate"
	"Gurl-cli/internal/transport"
)

type core struct{
	log *zap.Logger
	cfgPath string
	ic bool

	parser *config.Parser
	http *transport.HTTPClient
	grpc *transport.GRPCClient
}

func prettyPrint(data any, indent string) {
	switch v := any(data).(type) {
	case map[string]any:
		fmt.Println(indent+"[")
		for key, val := range v {
			switch valTyped := val.(type) {
			case map[string]any, []any:
				fmt.Printf("\n%s    %s:", indent, key)
				prettyPrint(valTyped, indent + "    ")
			default:
				fmt.Printf("\n%s    %s: %v", indent, key, valTyped)
			}
		}
		fmt.Println(indent+"]")
	case []any:
		for _, elem := range v {
			prettyPrint(elem, indent+"    ")
		}
	default:
		fmt.Printf("\n%s%v", indent, v)
	}
}

func (c *core) handleHTTP(cfg *config.HTTPConfig) {
	var err error
	var res transport.Result
	switch cfg.Method {
	case "GET":
		res, err = c.http.Get(cfg)
	case "POST":
		res, err = c.http.Post(cfg)
	case "PUT":
		res, err = c.http.Put(cfg)
	case "DELETE":
		res, err = c.http.Del(cfg)
	}
	if err != nil {
		c.log.Error("Make http request error",
			zap.String("Method", cfg.Method),
			zap.Error(err))
		return
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
	cfg.SetCookies(map[string][]*http.Cookie{
		res.URL.String(): res.Raw.Cookies(),
	})
	c.parser.ConfigUpd(cfg, c.cfgPath)
}

func (c *core) handleGRPC(cfg *config.GRPCConfig) {
	res, err := c.grpc.GRPC(cfg)
	if err != nil {
		c.log.Error("Make gRPC request error", zap.Error(err))
		return
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

		switch v := parsed.(type) {
		case *config.HTTPConfig:
			c.handleHTTP(v)
		case *config.GRPCConfig:
			c.handleGRPC(v)
		default:
			c.log.Error("Invalid config type", zap.Any("type", v.GetType()))
		}
	}
}

func Start(cfgType, cfgPath string, cfgCreate, ic bool, ckPath string, log *zap.Logger) {
	c := &core{
		log: log,
		cfgPath: cfgPath,
		ic: ic,
		parser: config.NewParser(log),
		http: transport.NewHTTP(ic, ckPath, log),
		grpc: transport.NewGRPC(log),
	}
	defer func(){
		if ckPath != "" && !cfgCreate { c.http.CkCl.SaveCookies() }
	}()
	if !cfgCreate {
		switch cfgType {
		case "http", "grpc", "repeated":
			c.handleRequest()
			return
		default:
			log.Fatal("Invalid config type", zap.String("config type", cfgType))
		}
	}
	if err := gen.InitConfig(cfgPath, cfgType, log); err != nil {
		log.Fatal("Generate config error", zap.Error(err))
	}
}
